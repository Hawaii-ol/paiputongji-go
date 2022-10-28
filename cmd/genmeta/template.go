package main

const RPCMAP_TEMPLATE = `// Auto generated, DO NOT EDIT.

package liqi

import 	"google.golang.org/protobuf/reflect/protoreflect"

type RPCDescriptor struct {
	Name string
	RequestType protoreflect.MessageType
	ResponseType protoreflect.MessageType
}

var RPCMap = map[string]*RPCDescriptor{
	{{- range $_, $item := . }}
	"{{ .RPCname }}": {
		"{{ .RPCname }}",
		(*{{ .ReqTypeName }})(nil).ProtoReflect().Type(),
		(*{{ .RespTypeName }})(nil).ProtoReflect().Type(),
	},
	{{- end }}
}
`

type TemplateDataItem struct {
	RPCname      string
	ReqTypeName  string
	RespTypeName string
}
