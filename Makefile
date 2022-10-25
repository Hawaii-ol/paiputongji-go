BLDPATH=build
CMDPATH=cmd
ifeq ($(OS), Windows_NT)
	EXEEXT=.exe
else
	EXEEXT=
endif

main:
	go build -o $(BLDPATH)/$@$(EXEEXT) $(CMDPATH)/$@/$@.go

genmeta:
	go build -o $(BLDPATH)/$@$(EXEEXT) $(CMDPATH)/$@/$@.go
	@$(BLDPATH)/$@$(EXEEXT)
	@mv liqi/liqi.pb $(BLDPATH)

goinstall:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

all: goinstall genmeta main

.PHONY: clean
clean:
	@rm -rf $(BLDPATH)/ liqi/liqi.proto
