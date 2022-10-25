package paiputongji

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"paiputongji/liqi"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MajsoulWSMsg struct {
	Type   string
	Time   float64
	Opcode int
	Data   string
}

type AccountBrief struct {
	AccountId uint32
	Nickname  string
}

type harStruct struct {
	Log struct {
		Entries []struct {
			WSMsgs []MajsoulWSMsg `json:"_webSocketMessages"`
		} `json:"entries"`
	} `json:"log"`
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

func recordGameToPaipu(rg *liqi.RecordGame, ab *AccountBrief) *Paipu {
	uuid := rg.Uuid
	if ab != nil {
		uuid += fmt.Sprintf("_a%d", EncryptAccountId(ab.AccountId))
	}
	/*	只统计四名人类玩家的友人场牌局，跳过三麻，AI玩家等
		GameConfig.category 1=友人场 2=段位场
		跳过修罗之战，血流换三张，瓦西子麻将等活动模式
	*/
	rule := rg.Config.Mode.DetailRule
	if rg.Config.Category == 1 &&
		len(rg.Accounts) == 4 &&
		rule.JiuchaoMode == 0 && // 瓦西子麻将
		rule.MuyuMode == 0 && // 龙之目玉
		rule.Xuezhandaodi == 0 && // 修罗之战
		rule.Huansanzhang == 0 && // 换三张
		rule.Chuanma == 0 && // 川麻血战到底
		rule.RevealDiscard == 0 && // 暗夜之战
		rule.FieldSpellMode == 0 { // 幻境传说
		paipu := Paipu{
			Uuid: uuid,
			Time: time.Unix(int64(rg.EndTime), 0),
		}
		var seats [4]*liqi.RecordGame_AccountInfo
		for _, account := range rg.Accounts {
			seats[account.Seat] = account
		}
		for i, player := range rg.Result.Players {
			account := seats[player.Seat]
			// part_point_1字段才是最后得分，total_point不知道是什么，可能是换算后的马点？
			paipu.Result[i] = PlayerScore{account.Nickname, int(player.PartPoint_1)}
		}
		return &paipu
	}
	return nil
}

// 解析HAR文件，获取牌谱列表
func ParseHARToPaipu(harpath string) ([]Paipu, *AccountBrief, error) {
	fp, err := os.Open(harpath)
	if err != nil {
		return nil, nil, err
	}
	decoder := jsoniter.NewDecoder(bufio.NewReader(fp))
	var data harStruct
	if err = decoder.Decode(&data); err != nil {
		return nil, nil, err
	}
	var account *AccountBrief
	paipuSlice := make([]Paipu, 0, 10)
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
							account = &AccountBrief{
								resLogin.AccountId,
								resLogin.Account.Nickname,
							}
						case "fetchGameRecordList": // 获取牌谱列表
							resGRList := new(liqi.ResGameRecordList)
							if err := proto.Unmarshal(wrapper.Data, resGRList); err != nil {
								return nil, nil, err
							}
							for _, record := range resGRList.RecordList {
								// 去除重复请求的牌谱
								if !uuidSet[record.Uuid] {
									uuidSet[record.Uuid] = true
									paipu := recordGameToPaipu(record, account)
									if paipu != nil {
										paipuSlice = append(paipuSlice, *paipu)
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

// 解析HAR文件，通过加载pb文件自动构造消息类型
func ParseHARDynamicpb(harpath string) error {
	fp, err := os.Open(harpath)
	if err != nil {
		return err
	}
	decoder := jsoniter.NewDecoder(bufio.NewReader(fp))
	var data harStruct
	if err = decoder.Decode(&data); err != nil {
		return err
	}
	fd, err := RegistryDescriptor()
	if err != nil {
		return err
	}
	lobby := fd.Services().ByName("Lobby")
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
					return err
				}
				msgType, msgId, wrapper, err := retriveMsgTypeIdProto(payload)
				if err != nil {
					return err
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
						rpcname := protoreflect.Name(tokens[len(tokens)-1])
						md := lobby.Methods().ByName(rpcname)
						if md == nil {
							return errors.New(
								fmt.Sprintf("Method %s does not exist in Lobby service", md.FullName()))
						}
						reqType, resType := md.Input(), md.Output()
						// TODO 现在我们只用到了牌谱列表这一个功能，所以只需要处理一种返回类型
						// 通过反射动态构造返回类型对应消息的方式可供日后开发
						log.Printf("Method %s(%s) -> %s\n", md.FullName(), reqType.Name(), resType.Name())
					} else {
						return errors.New(
							fmt.Sprintf("Response id %d does not exist in requests map", msgId))
					}
				}
			}
			break
		}
	}
	return nil
}
