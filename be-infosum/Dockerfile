FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-infosum
RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-infosum  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-infosum/bin /app

WORKDIR /app

EXPOSE 19090

VOLUME /data/conf

CMD ["./be-infosum", "--config", "/data/conf/config.yaml"]