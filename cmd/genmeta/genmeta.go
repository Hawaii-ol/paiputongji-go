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
	if compiler, err := gen.CompilerExecutable(); err != nil {
		log.Fatal(err)
	} else {
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
	}
}
