package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"paiputongji/gen"
)

func genProtoFile() {
	var conv gen.JsonToProtoConvertor
	jsonFile, err := os.Open(gen.JsonPath)
	if err != nil {
		log.Fatal(err)
	}
	protoFile, err := os.Create(gen.ProtoPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		jsonFile.Close()
		protoFile.Close()
		if p := recover(); p != nil {
			os.Remove(gen.ProtoPath)
			panic(p)
		}
	}()
	conv.Convert(jsonFile, protoFile)
}

func main() {
	// Generate `liqi.proto` file if not exists first
	if !gen.CheckProtoExists() {
		genProtoFile()
	}
	if compiler, err := gen.CompilerExecutable(); err != nil {
		log.Fatal(err)
	} else {
		// generate liqi.pb.go
		cmd := exec.Command(compiler,
			fmt.Sprintf("-I=%s", gen.RootDir),
			fmt.Sprintf("--go_out=%s", gen.RootDir),
			gen.ProtoPath,
		)
		fmt.Printf("Compiling %s ...\n", gen.META_FILENAME)
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
		fmt.Printf("Compiling descriptor %s ...\n", gen.DESCRIPTOR_FILENAME)
		out, err = cmd.CombinedOutput()
		os.Stdout.Write(out)
		if err != nil {
			log.Fatal("Compilation failed with ", err)
		}
	}
}
