package client

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"log"
	"os"
	"paiputongji/liqi"
)

// HandleNotification 解析服务器通知，执行相应动作
func HandleNotification(data []byte) error {
	wrapper := new(liqi.Wrapper)
	var err error
	if err = proto.Unmarshal(data, wrapper); err != nil {
		return err
	}
	// 通知没有方法名，name就是消息名，例如.lq.NotifyAccountUpdate
	// remove the leading dot
	name := protoreflect.FullName(wrapper.Name[1:])
	log.Println("server notification:", name)
	msgType, err := protoregistry.GlobalTypes.FindMessageByName(name)
	if err != nil {
		return err
	}
	message := msgType.New().Interface()
	if err = proto.Unmarshal(wrapper.Data, message); err != nil {
		return err
	}
	handler := switchNotifHandler(message)
	handler(message)
	return nil
}

func switchNotifHandler(message interface{}) func(interface{}) {
	switch message.(type) {
	case *liqi.NotifyAnotherLogin:
		return handleNotifyAnotherLogin
	default:
		return defaultHandler
	}
}

func handleNotifyAnotherLogin(message interface{}) {
	fmt.Println("您的账号已在另一处登录")
	fmt.Println("请注意不要在使用本软件时登录游戏。为避免冲突，程序即将退出。")
	os.Exit(0)
}

func defaultHandler(message interface{}) {
	fmt.Println(message)
	log.Println("notification content:", message)
}
