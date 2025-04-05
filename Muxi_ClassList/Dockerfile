FROM golang:1.22 AS builder


COPY ../.. /src
WORKDIR /src

RUN GOPROXY=https://goproxy.cn,direct make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

COPY --from=builder /src/configs /app/configs
WORKDIR /app

EXPOSE 19087


## 注意，这里的“class”是main.go所在的目录名（猜的）
CMD ["./class"]
