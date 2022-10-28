package paiputongji

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"paiputongji/liqi"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/proto"
)

type harStruct struct {
	Log struct {
		Entries []struct {
			WSMsgs []HarWSMessageJson `json:"_webSocketMessages"`
		} `json:"entries"`
	} `json:"log"`
}

type HarWSMessageJson struct {
	Type   string
	Time   float64
	Opcode int
	Data   string
}

const (
	MSG_NOTIFICATION = 0x01
	MSG_REQUEST      = 0x02
	MSG_RESPONSE     = 0x03
)

func retriveMsgTypeIdProto(payload []byte) (msgType int, msgId int, wrapper *liqi.Wrapper, err error) {
	msgType = int(payload[0])
	switch msgType {
	case MSG_NOTIFICATION:
		// do something
	case MSG_REQUEST, MSG_RESPONSE:
		msgId = int(binary.LittleEndian.Uint16(payload[1:3]))
		wrapper = new(liqi.Wrapper)
		err = proto.Unmarshal(payload[3:], wrapper)
	default:
		err = errors.New(fmt.Sprintf("Unknown message BOM 0x%x", msgType))
	}
	return
}

// 解析HAR文件，获取牌谱列表
func ParseHARToPaipu(harpath string) ([]*Paipu, *liqi.Account, error) {
	fp, err := os.Open(harpath)
	if err != nil {
		return nil, nil, err
	}
	decoder := jsoniter.NewDecoder(bufio.NewReader(fp))
	var data harStruct
	if err = decoder.Decode(&data); err != nil {
		return nil, nil, err
	}
	var account *liqi.Account
	paipuSlice := make([]*Paipu, 0, 10)
	uuidSet := make(map[string]bool)
	reqMap := make(map[int]*liqi.Wrapper)
	for _, entry := range data.Log.Entries {
		if entry.WSMsgs != nil {
			/*	报文的首字节决定消息类型
				0x01 服务器主动推送的通知
				0x02 客户端请求
				0x03 服务器响应
				对于0x02和0x03类型的报文，第2~3字节是报文的id，确保请求和响应正确对应；第4字节起才是正文
				正文是定义在liqi.proto中的Wrapper类型消息
			*/
			for _, msg := range entry.WSMsgs {
				payload, err := base64.StdEncoding.DecodeString(msg.Data)
				if err != nil {
					return nil, nil, err
				}
				msgType, msgId, wrapper, err := retriveMsgTypeIdProto(payload)
				if err != nil {
					return nil, nil, err
				}
				switch msgType {
				case MSG_NOTIFICATION:
					// do something
				case MSG_REQUEST:
					reqMap[msgId] = wrapper
				case MSG_RESPONSE:
					if reqWrapper, ok := reqMap[msgId]; ok {
						/*	响应报文没有name，需要通过请求报文的id判断响应类型
							name形如：
							.lq.Lobby.heatbeat
							.lq.Lobby.oauth2Login
							.lq.Lobby.fetchFriendList
							通过尾部的RPC方法名查找liqi.proto中方法对应的返回类型
						*/
						tokens := strings.Split(reqWrapper.Name, ".")
						rpcname := tokens[len(tokens)-1]
						switch rpcname {
						case "oauth2Login": // OAUTH2登录
							resLogin := new(liqi.ResLogin)
							if err := proto.Unmarshal(wrapper.Data, resLogin); err != nil {
								return nil, nil, err
							}
							account = resLogin.Account
						case "fetchGameRecordList": // 获取牌谱列表
							resGRList := new(liqi.ResGameRecordList)
							if err := proto.Unmarshal(wrapper.Data, resGRList); err != nil {
								return nil, nil, err
							}
							for _, record := range resGRList.RecordList {
								// 去除重复请求的牌谱
								if !uuidSet[record.Uuid] {
									uuidSet[record.Uuid] = true
									paipu := RecordGameToPaipu(record, account)
									if paipu != nil {
										paipuSlice = append(paipuSlice, paipu)
									}
								}
							}
						}
					} else {
						return nil, nil, errors.New(
							fmt.Sprintf("Response id %d does not exist in requests map", msgId))
					}
				}
			}
			break
		}
	}
	return paipuSlice, account, nil
}
