FROM golang:1.23.7 AS builder

COPY . /src

WORKDIR /src/be-static
RUN go mod tidy

RUN GOPROXY=https://goproxy.cn go build -o ./bin/be-static  ./...

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-static/bin /app

WORKDIR /app

EXPOSE 19091

VOLUME /data/conf

CMD ["./be-static", "-config", "/data/conf/config.yaml"]