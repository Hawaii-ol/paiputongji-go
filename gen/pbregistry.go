package gen

import (
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// protoc生成的Go代码不包含Service部分，需要另外设法加载
// 这个函数从protoc -o选项生成的pb文件中加载全部的Descriptor
// 需要运行前置任务make genmeta生成pb文件
func RegisterDescriptor(pbFilePath string) (protoreflect.FileDescriptor, error) {
	pbData, err := os.ReadFile(pbFilePath)
	if err != nil {
		return nil, err
	}
	fdset := descriptorpb.FileDescriptorSet{}
	if err = proto.Unmarshal(pbData, &fdset); err != nil {
		return nil, err
	}
	fdproto, resolver := fdset.GetFile()[0], new(protoregistry.Files)
	return protodesc.NewFile(fdproto, resolver)
}
