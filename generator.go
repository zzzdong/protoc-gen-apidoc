package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MethodDef struct {
	*protogen.Method
	httpMethod string
	httpPath   string
}

func GenerateFile(gen *protogen.Plugin, file *protogen.File) (*protogen.GeneratedFile, error) {
	filename := file.GeneratedFilenamePrefix + ".apidoc.md"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	// g.P("// Code generated by protoc-gen-apidoc.")
	// g.P("// source: ", file.Desc.Path())
	// g.P()

	methods := getAllMethods(file)

	g.P(file.GeneratedFilenamePrefix, " API DOC")
	g.P("======")
	g.P("[toc]")

	for _, method := range methods {
		g.P("## ", string(method.Desc.FullName()))
		g.P()
		g.P(string(method.Comments.Leading))

		// path info
		g.P("`", method.httpMethod, " ", method.httpPath, "`")
		g.P()

		g.P("### Request")
		g.P()
		printMessage(g, method.Input)
		g.P()

		if method.httpMethod != "GET" {
			g.P("#### JSON")
			g.P()
			printMessageJson(g, method.Input)
			g.P()
		}

		g.P("### Response payload")
		g.P()
		printMessage(g, method.Output)
		g.P()
		g.P("#### JSON")
		g.P()
		printWrappedMessageJson(g, method.Output)
		g.P()
	}

	return g, nil
}

func getAllMethods(file *protogen.File) []*MethodDef {
	methods := make([]*MethodDef, 0)

	for _, svc := range file.Services {
		for _, method := range svc.Methods {
			if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
				continue
			}

			methods = append(methods, parseMethod(method))
		}
	}

	return methods
}

func parseMethod(m *protogen.Method) *MethodDef {
	httpMethod := "POST"
	httpPath := string(m.Desc.FullName())

	if opts, ok := m.Desc.Options().(*descriptorpb.MethodOptions); ok {
		if httpRule, ok := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule); ok {
			switch httpRule.GetPattern().(type) {
			case *annotations.HttpRule_Get:
				httpMethod = "GET"
				httpPath = httpRule.GetGet()
			case *annotations.HttpRule_Put:
				httpMethod = "PUT"
				httpPath = httpRule.GetPut()
			case *annotations.HttpRule_Post:
				httpMethod = "POST"
				httpPath = httpRule.GetPost()
			case *annotations.HttpRule_Delete:
				httpMethod = "DELETE"
				httpPath = httpRule.GetDelete()
			case *annotations.HttpRule_Patch:
				httpMethod = "DELETE"
				httpPath = httpRule.GetPatch()
			default:
			}
		}
	}

	method := &MethodDef{Method: m, httpMethod: httpMethod, httpPath: httpPath}

	return method
}

func printMessage(g *protogen.GeneratedFile, message *protogen.Message) {
	// table header
	g.P("| 字段        | 类型        | 描述  |")
	g.P("| ----------- |:----------:| -----|")

	for _, field := range message.Fields {
		printField(g, 0, "", field)
	}
}

func printField(g *protogen.GeneratedFile, indent int, name string, field *protogen.Field) {
	fieldName := field.Desc.JSONName()
	if indent > 0 {
		fieldName = fmt.Sprintf("%s %s.%s", strings.Repeat(">", indent), name, fieldName)
	}

	kind := field.Desc.Kind().String()
	if field.Desc.IsList() {
		kind = "array"
	}

	row := fmt.Sprintf("|%s|%s|%s|", fieldName, kind, getCompactComment(&field.Comments))
	g.P(row)

	if field.Message != nil && field.Message.Desc.FullName() != field.Parent.Desc.FullName() {
		printSubMessage(g, indent+1, field.Desc.JSONName(), field.Message)
	}
}

func printSubMessage(g *protogen.GeneratedFile, indent int, name string, message *protogen.Message) {
	for _, field := range message.Fields {
		printField(g, indent, name, field)
	}

}

func printWrappedMessageJson(g *protogen.GeneratedFile, message *protogen.Message) {
	msg := messageToInterface(message)

	type WrappedResp struct {
		ErrCode int32       `json:"errCode"`
		ErrMsg  string      `json:"errMsg"`
		Data    interface{} `json:"data"`
	}

	resp := WrappedResp{
		ErrCode: 0,
		ErrMsg:  "ok",
		Data:    msg,
	}

	bs, _ := json.MarshalIndent(&resp, "", "\t")

	g.P("```json")
	g.P(string(bs))
	g.P("```")
}

func printMessageJson(g *protogen.GeneratedFile, message *protogen.Message) {
	msg := messageToInterface(message)

	bs, _ := json.MarshalIndent(msg, "", "\t")

	g.P("```json")
	g.P(string(bs))
	g.P("```")
}

func messageToInterface(message *protogen.Message) interface{} {
	st := make(map[string]interface{})

	for _, field := range message.Fields {
		var fi interface{}

		if field.Message != nil && field.Message.Desc.FullName() == message.Desc.FullName() {
			fi = map[string]interface{}{}
			if field.Desc.IsList() {
				fi = []interface{}{}
			}
		} else {
			if field.Desc.IsList() {
				item := fieldToInterface(field)
				fi = []interface{}{item}
			} else if field.Desc.IsMap() {
				fi = map[string]interface{}{}
			} else {
				fi = fieldToInterface(field)
			}
		}

		st[string(field.Desc.JSONName())] = fi
	}

	return st
}

func fieldToInterface(field *protogen.Field) interface{} {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return false

	case protoreflect.EnumKind:
		return 0

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind:
		return int(0)

	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind, protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		return float32(0.0)

	case protoreflect.StringKind, protoreflect.BytesKind:
		return string("")
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageToInterface(field.Message)
	}

	return map[string]interface{}{}
}

func getCompactComment(comment *protogen.CommentSet) string {
	text := strings.Trim(comment.Leading.String(), "/ \t\r\n")

	lines := strings.Split(text, "\n")

	newLines := make([]string, 0)
	for _, line := range lines {
		line := strings.Trim(line, "/ \t\r\n")
		newLines = append(newLines, line)
	}
	c := strings.Join(newLines, " ")

	c += strings.Trim(comment.Trailing.String(), "/ \t\r\n")

	return c
}
