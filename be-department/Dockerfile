FROM golang:1.24 AS builder

COPY . /src

WORKDIR /src/be-department

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod tidy

RUN go build -o ./bin/be-department  .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src/be-department/bin /app

WORKDIR /app

EXPOSE 19086

VOLUME /data/conf

CMD ["./be-department", "--config", "/data/conf/config.yaml"]