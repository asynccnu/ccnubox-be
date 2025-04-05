GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
#	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find proto -name *.proto")
else
#	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find proto -name *.proto)
endif


.PHONY: api
# 为所有proto文件生成proto.go文件
api:
	protoc --proto_path=./proto \
           --proto_path=./third_party \
           --experimental_allow_proto3_optional \
           --go-errors_out=paths=source_relative:./gen/proto \
 	       --go_out=paths=source_relative:./gen/proto \
 	       --go-http_out=paths=source_relative:./gen/proto \
 	       --go-grpc_out=paths=source_relative:./gen/proto \
	       $(API_PROTO_FILES)
