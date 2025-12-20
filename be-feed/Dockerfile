FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-feed
RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-feed  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-feed/bin /app

WORKDIR /app

EXPOSE 19088

VOLUME /data/conf

CMD ["./be-feed", "--config", "/data/conf/config.yaml"]