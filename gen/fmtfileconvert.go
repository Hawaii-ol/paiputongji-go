package gen

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type liqiRootWrapper struct {
	Nested struct {
		Lq struct {
			Nested map[string]interface{}
		}
	}
}

type protoField struct {
	rule  string
	name  string
	_type string
	id    int
}

type protoEnum struct {
	name  string
	value int
}

type JsonToProtoConvertor struct {
	indent int
	writer io.Writer
}

func (conv JsonToProtoConvertor) writeIndentedLine(content string) {
	whitespaces := strings.Repeat(" ", conv.indent*2)
	conv.writer.Write([]byte(whitespaces + content + "\n"))
}

func (conv JsonToProtoConvertor) Convert(jsonFile io.Reader, protoFile io.Writer) error {
	conv.indent = 0
	conv.writer = protoFile
	// write syntax info
	conv.writeIndentedLine("syntax = \"proto3\";\n")
	// write package name
	conv.writeIndentedLine("package lq;\n")
	// write option go_package
	conv.writeIndentedLine("option go_package = \"liqi/\";\n")
	var root liqiRootWrapper
	decoder := jsoniter.NewDecoder(jsonFile)
	if err := decoder.Decode(&root); err != nil {
		return err
	}
	// map returns keys in a random order, need to sort them first
	definitions := root.Nested.Lq.Nested
	keys := sortedMapKeys(definitions)
	// iterate over package and parse each item
	for _, name := range keys {
		conv.parseItem(name, definitions[name].(map[string]interface{}))
	}
	return nil
}

func sortedMapKeys(m map[string]interface{}) []string {
	i, slice := 0, make([]string, len(m))
	for key := range m {
		slice[i] = key
		i++
	}
	sort.Strings(slice)
	return slice
}

func parseProtoField(name string, props map[string]interface{}) protoField {
	field := protoField{
		name:  name,
		_type: props["type"].(string),
		id:    int(props["id"].(float64)),
	}
	if rule := props["rule"]; rule != nil {
		field.rule = rule.(string)
	}
	return field
}

func (conv JsonToProtoConvertor) parseItem(name string, item map[string]interface{}) {
	if fields, ok := item["fields"]; ok {
		// check for fields(=message)
		conv.writeIndentedLine(fmt.Sprintf("message %s {", name))
		conv.indent++
		fieldMap := fields.(map[string]interface{})
		// sort by field id first
		i, fieldSlice := 0, make([]protoField, len(fieldMap))
		for name, props := range fieldMap {
			fieldSlice[i] = parseProtoField(name, props.(map[string]interface{}))
			i++
		}
		sort.Slice(fieldSlice, func(i, j int) bool {
			return fieldSlice[i].id < fieldSlice[j].id
		})
		for _, field := range fieldSlice {
			if field.rule == "" {
				// type varname = id
				conv.writeIndentedLine(fmt.Sprintf(
					"%s %s = %d;", field._type, field.name, field.id))
			} else {
				// rule type varname = id
				conv.writeIndentedLine(fmt.Sprintf(
					"%s %s %s = %d;", field.rule, field._type, field.name, field.id))
			}
		}
	} else if methods, ok := item["methods"]; ok {
		// check for methods(=service)
		conv.writeIndentedLine(fmt.Sprintf("service %s {", name))
		conv.indent++
		methodMap := methods.(map[string]interface{})
		keys := sortedMapKeys(methodMap)
		for _, key := range keys {
			props := methodMap[key].(map[string]interface{})
			// rpc methodName (requestType) returns (responseType)
			conv.writeIndentedLine(fmt.Sprintf(
				"rpc %s (%s) returns (%s);",
				key,
				props["requestType"].(string),
				props["responseType"].(string),
			))
		}
	} else if values, ok := item["values"]; ok {
		// check for values(=enum)
		conv.writeIndentedLine(fmt.Sprintf("enum %s {", name))
		conv.indent++
		valueMap := values.(map[string]interface{})
		// sort by enum value
		i, valueSlice := 0, make([]protoEnum, len(valueMap))
		for name, value := range valueMap {
			valueSlice[i] = protoEnum{name, int(value.(float64))}
			i++
		}
		sort.Slice(valueSlice, func(i, j int) bool {
			return valueSlice[i].value < valueSlice[j].value
		})
		for _, enum := range valueSlice {
			// NAME = VALUE
			conv.writeIndentedLine(fmt.Sprintf("%s = %d;", enum.name, enum.value))
		}
	} else {
		// unknown new type
		log.Printf("Unknown new type \"%s\" = %v", name, item)
	}

	// parse child items recursively
	if nested, ok := item["nested"]; ok {
		nestedMap := nested.(map[string]interface{})
		keys := sortedMapKeys(nestedMap)
		for _, key := range keys {
			conv.parseItem(key, nestedMap[key].(map[string]interface{}))
		}
	}

	conv.indent--
	conv.writeIndentedLine("}")
	if conv.indent == 0 {
		conv.writer.Write([]byte("\n"))
	}
}
