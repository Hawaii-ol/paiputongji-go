package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"paiputongji/gen"
)

func genProtoFile() error {
	var conv gen.JsonToProtoConvertor
	jsonFile, err := os.Open(gen.JsonPath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	protoFile, err := os.Create(gen.ProtoPath)
	if err != nil {
		return err
	}
	defer protoFile.Close()
	return conv.Convert(jsonFile, protoFile)
}

func main() {
	// Always re-generate `liqi.proto` to make sure it's the latest
	fmt.Printf("Converting %s to %s...\n", gen.JSON_FILENAME, gen.PROTO_FILENAME)
	if err := genProtoFile(); err != nil {
		log.Fatal(err)
	}
	compiler, err := gen.CompilerExecutable()
	if err != nil {
		log.Fatal(err)
	}
	// generate liqi.pb.go
	cmd := exec.Command(compiler,
		fmt.Sprintf("-I=%s", gen.RootDir),
		fmt.Sprintf("--go_out=%s", gen.RootDir),
		gen.ProtoPath,
	)
	fmt.Printf("Compiling %s...\n", gen.META_FILENAME)
	out, err := cmd.CombinedOutput()
	os.Stdout.Write(out)
	if err != nil {
		log.Fatal("Compilation failed with ", err)
	}
	// generate liqi.pb descriptor
	cmd = exec.Command(compiler,
		fmt.Sprintf("-I=%s", gen.RootDir),
		fmt.Sprintf("-o%s", gen.DescriptorPath),
		gen.ProtoPath,
	)
	fmt.Printf("Compiling descriptor %s...\n", gen.DESCRIPTOR_FILENAME)
	out, err = cmd.CombinedOutput()
	os.Stdout.Write(out)
	if err != nil {
		log.Fatal("Compilation failed with ", err)
	}
	// load descriptor file and generate RPCMap
	/*
		fmt.Printf("Loading %s to generate %s...\n", gen.DESCRIPTOR_FILENAME, gen.RPCMAP_FILENAME)
		fd, err := gen.RegisterDescriptor(gen.DescriptorPath)
		if err != nil {
			log.Fatalf("Failed to load %s.\n", gen.DESCRIPTOR_FILENAME)
		}
		lobby := fd.Services().ByName("Lobby")
		methods := lobby.Methods()
		tmplData := make([]TemplateDataItem, methods.Len())
		for i := 0; i < methods.Len(); i++ {
			m := methods.Get(i)
			reqTypeName := strings.Title(string(m.Input().Name()))
			respTypeName := strings.Title(string(m.Output().Name()))
			tmplData[i] = TemplateDataItem{string(m.Name()), reqTypeName, respTypeName}
		}
		t := template.Must(template.New(gen.RPCMAP_FILENAME).Parse(RPCMAP_TEMPLATE))
		rpcfile, err := os.Create(gen.RPCMapPath)
		if err != nil {
			log.Fatal(err)
		}
		if err := t.Execute(rpcfile, tmplData); err != nil {
			log.Fatal(err)
		}
	*/
}
