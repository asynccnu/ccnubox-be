FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-class

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-class ./cmd/class

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/be-class/bin /app

WORKDIR /app

EXPOSE 18000
EXPOSE 19083

VOLUME /data/conf

CMD ["./be-class", "-conf", "/data/conf/config.yaml"]
