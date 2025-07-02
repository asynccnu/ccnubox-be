FROM golang:1.23.7 AS builder

COPY . /src

WORKDIR /src/be-library

ENV GOPROXY=https://goproxy.cn,direct

RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-library ./cmd/be-library

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-library/bin /app

WORKDIR /app

EXPOSE 19093

VOLUME /data/conf

CMD ["./be-library", "--conf", "/data/conf/config.yaml"]