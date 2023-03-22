package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"paiputongji/liqi"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	MSG_NOTIFICATION = 0x01
	MSG_REQUEST      = 0x02
	MSG_RESPONSE     = 0x03
)

type MajsoulWSClient struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	reqId      int
	reqMap     map[int]proto.Message
	reqChanMap map[int]chan []byte // 回调channel，请求发送后阻塞在此channel上等待响应
	Api        ApiDelegate
	Account    *liqi.Account
}

func NewMajsoulClient() *MajsoulWSClient {
	cli := &MajsoulWSClient{
		reqId:      1,
		reqMap:     make(map[int]proto.Message),
		reqChanMap: make(map[int]chan []byte),
	}
	cli.Api.cli = cli
	return cli
}

func (cli *MajsoulWSClient) Connect() error {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}
	header := make(http.Header)
	header.Set("User-Agent", USER_AGENT)
	header.Set("Origin", MAJSOUL_URLBASE[:len(MAJSOUL_URLBASE)-3]) // remove the trailing /1/
	conn, _, err := dialer.Dial(MAJSOUL_GATEWAY, header)
	if err != nil {
		return err
	}
	cli.conn = conn
	return nil
}

func (cli *MajsoulWSClient) IsLogin() bool {
	return cli.Account != nil
}

// 注册请求，返回分配的请求id
func (cli *MajsoulWSClient) registerRequest(request proto.Message, callback chan []byte) int {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	id := cli.reqId
	cli.reqMap[id] = request
	cli.reqChanMap[id] = callback
	cli.reqId++
	return id
}

// 注销客户端中的请求记录，返回原始请求和对应的回调channel
func (cli *MajsoulWSClient) checkoutRequest(reqId int) (proto.Message, chan []byte, error) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	request, ok := cli.reqMap[reqId]
	if !ok {
		return nil, nil, errors.New(fmt.Sprintf("unregistered request id: %d\n", reqId))
	}
	callback := cli.reqChanMap[reqId]
	delete(cli.reqMap, reqId)
	delete(cli.reqChanMap, reqId)
	return request, callback, nil
}

/*
使用liqi.Wrapper封装一个报文
报文的首字节决定消息类型
0x01 服务器主动推送的通知
0x02 客户端请求
0x03 服务器响应
对于0x01类型的报文，第2字节起是正文
对于0x02和0x03类型的报文，第2~3字节是报文的id，确保请求和响应正确对应；第4字节起才是正文
正文是定义在liqi.proto中的Wrapper类型消息
*/
func wrapMessage(msgType int, msgId int, rpcname string, message proto.Message) ([]byte, error) {
	// 天坑！雀魂的服务器/客户端使用的通信格式没有完全遵守proto3！
	// proto3协议规定如果一个optional字段是0值，那么它等同于未设置，可以不被序列化。
	// 例如一条proto message有三个uint32字段，其中有两个是0值，那么序列化后的字节流中将只包含一个非0字段的tag和值
	//
	// 而雀魂的通信格式不能省略字段！如果使用标准的golang/proto包序列化一条上述消息，那么雀魂的服务器将无法识别！
	// 必须完整地传送三个字段，哪怕其中某些字段是0值
	var data []byte
	var err error
	if patchedMarshal := rpcMarshalNeedPatch[rpcname]; patchedMarshal != nil {
		data, err = patchedMarshal(message)
	} else {
		data, err = proto.Marshal(message)
	}
	if err != nil {
		return nil, err
	}
	wrapper := &liqi.Wrapper{Name: ".lq.Lobby." + rpcname, Data: data}
	if data, err = proto.Marshal(wrapper); err != nil {
		return nil, err
	}
	var b []byte
	switch msgType {
	case MSG_NOTIFICATION:
		b = make([]byte, 1+len(data))
		b[0] = byte(msgType)
		copy(b[1:], data)
	case MSG_REQUEST, MSG_RESPONSE:
		b = make([]byte, 3+len(data))
		b[0] = byte(msgType)
		binary.LittleEndian.PutUint16(b[1:], uint16(msgId))
		copy(b[3:], data)
	default:
		return nil, errors.New(fmt.Sprintf("unknown message type 0x%x", msgType))
	}
	return b, nil
}

func unwrapMessage(data []byte) (msgType int, msgId int, payload []byte, err error) {
	msgType = int(data[0])
	switch msgType {
	case MSG_NOTIFICATION:
		payload = data[1:]
	case MSG_REQUEST, MSG_RESPONSE:
		msgId = int(binary.LittleEndian.Uint16(data[1:3]))
		wrapper := new(liqi.Wrapper)
		if err = proto.Unmarshal(data[3:], wrapper); err != nil {
			return
		}
		payload = wrapper.Data
	default:
		err = errors.New(fmt.Sprintf("unknown message type 0x%x", msgId))
		return
	}
	return
}

// SendMessage 发送一条消息，callback为接收响应的回调channel
//
// goroutine-safe
func (cli *MajsoulWSClient) SendMessage(rpcname string, message proto.Message, callback chan []byte) (err error) {
	if cli.conn == nil {
		return errors.New("not connected yet, use Connect() first")
	}
	// check if message type matches rpc's request type
	/*
		rpcdesc := liqi.RPCMap[rpcname]
		if rpcdesc == nil {
			return errors.New(fmt.Sprintf("method %s does not exist in Lobby service", rpcname))
		}
		if message.ProtoReflect().Type() != rpcdesc.RequestType {
			msgType := string(message.ProtoReflect().Descriptor().Name())
			wantedType := string(rpcdesc.RequestType.Descriptor().Name())
			return errors.New(
				fmt.Sprintf(`invalid message type "%s" for method %s, expect "%s"`, msgType, rpcname, wantedType))
		}
	*/
	reqId := cli.registerRequest(message, callback)
	defer func() {
		if err != nil {
			// make sure we check out the registered id
			cli.checkoutRequest(reqId)
		}
	}()
	data, err := wrapMessage(MSG_REQUEST, reqId, rpcname, message)
	if err != nil {
		return err
	}
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return cli.conn.WriteMessage(websocket.BinaryMessage, data)
}

// SelectMessage 循环读取消息，并将响应发回对应的回调channel。
// 如果发生连接或读取错误将panic。其他错误例如数据错误将忽略并打印日志。
//
// goroutine-safe
func (cli *MajsoulWSClient) SelectMessage(wg *sync.WaitGroup) {
	defer wg.Done()
	if cli.conn == nil {
		panic(errors.New("not connected yet, use Connect() first"))
	}
	for {
		_, data, err := cli.conn.ReadMessage()
		if err != nil {
			cli.mu.Lock()
			activeClose := false
			// 切换用户等操作会主动关闭连接
			if cli.conn == nil {
				activeClose = true
			}
			cli.mu.Unlock()
			if activeClose {
				break
			} else {
				panic(err)
			}
		}
		msgType, msgId, payload, err := unwrapMessage(data)
		if err != nil {
			log.Println(err)
			continue
		}
		if msgType == MSG_NOTIFICATION {
			fmt.Print("接收到服务器通知: ")
			if err := HandleNotification(payload); err != nil {
				fmt.Println("解析通知内容失败:", err)
				log.Println("failed to handle notification:", err)
				log.Println("notification payload:", payload)
			}
		} else {
			_, callback, err := cli.checkoutRequest(msgId)
			if err != nil {
				log.Println(err)
				continue
			}
			callback <- payload
		}
	}
}

// StartHeartBeat 模拟心跳包，间隔为秒
func (cli *MajsoulWSClient) StartHeartBeat(intervalSec int, abort chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Second * time.Duration(intervalSec))
	defer ticker.Stop()
	for {
		if err := cli.Api.HeatBeat(); err != nil {
			return
		}
		select {
		case <-ticker.C:
			continue
		case <-abort:
			return
		}
	}
}

func (cli *MajsoulWSClient) Close() {
	if cli.conn != nil {
		cli.mu.Lock()
		defer cli.mu.Unlock()
		cli.conn.Close()
		cli.reqId = 0
		cli.reqMap = make(map[int]proto.Message)
		for id, callback := range cli.reqChanMap {
			close(callback)
			delete(cli.reqChanMap, id)
		}
		cli.conn = nil
	}
}
