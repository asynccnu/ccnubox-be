FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-user
RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-user  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai


# 声明构建时的变量
ARG KEY


ENV CRYPTO_KEY=${KEY}

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-user/bin /app

WORKDIR /app

EXPOSE 19091

VOLUME /data/conf

CMD ["./be-user", "--config", "/data/conf/config.yaml"]