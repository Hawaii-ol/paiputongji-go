package client

import (
	"paiputongji/liqi"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// 标准的proto.Marshall方法将会省略所有的0值字段
// 而雀魂的服务器不能识别这样的信息(但反过来标准库可以解析任何服务器信息)
// 因此必须重写marshall方法在序列化时保留特定字段

func patchReqGameRecordListMarshal(message proto.Message) ([]byte, error) {
	request := message.(*liqi.ReqGameRecordList)
	data := make([]byte, 0, 10)
	// append start field(type=uint32, id=1)
	data = protowire.AppendTag(data, 1, protowire.VarintType)
	data = protowire.AppendVarint(data, uint64(request.Start))
	// append count field(type=uint32, id=2)
	data = protowire.AppendTag(data, 2, protowire.VarintType)
	data = protowire.AppendVarint(data, uint64(request.Count))
	// append type field(type=uint32, id=3)
	data = protowire.AppendTag(data, 3, protowire.VarintType)
	data = protowire.AppendVarint(data, uint64(request.Type))
	return data, nil
}

var rpcMarshalNeedPatch = map[string]func(proto.Message) ([]byte, error){
	"fetchGameRecordList": patchReqGameRecordListMarshal,
}
