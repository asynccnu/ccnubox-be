FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-grade

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod tidy

RUN go build -o ./bin/be-grade  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-grade/bin /app

WORKDIR /app

EXPOSE 19089

VOLUME /data/conf

CMD ["./be-grade", "--config", "/data/conf/config.yaml"]