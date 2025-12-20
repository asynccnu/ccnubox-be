FROM golang:1.24 AS builder

ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on

COPY . /src

WORKDIR /src/bff
RUN go mod tidy
RUN go install github.com/swaggo/swag/cmd/swag@latest 
RUN go build -o ./bin/bff  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

# 安装运行依赖 + 构建 Swagger 所需工具
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        netbase \
        make \
        nodejs \
        npm \
    && npm install -g swagger2openapi \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/bff /app
COPY --from=builder /go/bin/swag /usr/local/bin/swag


WORKDIR /app

EXPOSE 8080

VOLUME /data/conf

CMD ["./bin/bff", "--config", "/data/conf/config.yaml"]