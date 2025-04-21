FROM golang:1.23.7 AS builder

COPY . /src

WORKDIR /src/be-banner
RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-banner  ./...

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-banner/bin /app

WORKDIR /app

EXPOSE 19080

VOLUME /data/conf

CMD ["./be-banner", "-config", "/data/conf/config.yaml"]