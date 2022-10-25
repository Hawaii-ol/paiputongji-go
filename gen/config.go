package gen

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const (
	JSON_FILENAME       = "liqi.json"
	PROTO_FILENAME      = "liqi.proto"
	META_FILENAME       = "liqi.pb.go"
	DESCRIPTOR_FILENAME = "liqi.pb"
)

var exe, _ = os.Executable()
var RootDir = filepath.Join(filepath.Dir(exe), "..") // program runs in build/
var JsonPath = filepath.Join(RootDir, "liqi", JSON_FILENAME)
var ProtoPath = filepath.Join(RootDir, "liqi", PROTO_FILENAME)
var DescriptorPath = filepath.Join(RootDir, "liqi", DESCRIPTOR_FILENAME)

func CompilerExecutable() (compiler string, err error) {
	// Determine the protoc compiler
	// relative path will NOT be executed by cmd.Run(), have to use absolute path.
	compilerDir, _ := filepath.Abs(filepath.Join(RootDir, "compiler"))
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			compiler = filepath.Join(compilerDir, "windows", "x64", "bin", "protoc.exe")
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			compiler = filepath.Join(compilerDir, "linux", "x64", "bin", "protoc")
		}
	case "darwin":
		compiler = filepath.Join(compilerDir, "osx", "universal", "bin", "protoc")
	}
	if compiler == "" {
		err = errors.New("Unable to find a proper compiler for your platform, or your platform cannot be determined")
	}
	return
}

func checkFileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		// treat all errnos as not exists
		return false
	} else {
		return true
	}
}

func CheckProtoExists() bool {
	return checkFileExists(ProtoPath)
}

func CheckMetafileExists() bool {
	return checkFileExists(filepath.Join(RootDir, "liqi", META_FILENAME))
}

func CheckDescriptorExists() bool {
	return checkFileExists(DescriptorPath)
}
