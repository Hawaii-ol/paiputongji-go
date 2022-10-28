BLDPATH=build
CMDPATH=cmd
ifeq ($(OS), Windows_NT)
	EXEEXT=.exe
else
	EXEEXT=
endif

all: goinstall genmeta main update

main:
	go build -o $(BLDPATH)/paiputongji$(EXEEXT) ./$(CMDPATH)/$@/
	@echo Copying resource files into $(BLDPATH)/...
	@cp -r res $(BLDPATH)/

genmeta:
	go build -o $(BLDPATH)/$@$(EXEEXT) ./$(CMDPATH)/$@/
	@$(BLDPATH)/$@$(EXEEXT)

goinstall:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

update:
	go build -o $(BLDPATH)/$@$(EXEEXT) ./$(CMDPATH)/$@/

.PHONY: clean
clean:
	@rm -rf $(BLDPATH) liqi/liqi.proto
