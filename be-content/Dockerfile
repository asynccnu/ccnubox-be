FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-content

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod tidy

RUN go build -o ./bin/be-content  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-content/bin /app

WORKDIR /app

EXPOSE 19081

VOLUME /data/conf

CMD ["./be-content", "--config", "/data/conf/config.yaml"]