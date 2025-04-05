# be-api

proto for grpc，**所有的proto都放在这个地方，集中进行管理，便于各个服务调用**

使用前请先进行前置依赖安装将相关依赖下载到gopath的bin目录下面

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/go-errors/errors/cmd/protoc-gen-go-errors@latest
```

