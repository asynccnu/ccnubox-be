FROM golang:1.23.7 AS builder

COPY . /src

WORKDIR /src/be-classlist

RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-classlist ./cmd/classlist

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/be-classlist/bin /app

WORKDIR /app

EXPOSE 19084
VOLUME /data/conf

## 注意，这里的“classlist”是main.go所在的目录名（猜的）
CMD ["./be-classlist", "-conf", "/data/conf/config.yaml"]
