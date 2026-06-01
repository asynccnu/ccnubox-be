FROM golang:1.26 AS builder

COPY . /src

WORKDIR /src/be-classlist_v2

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod tidy

RUN go build -o ./bin/be-classlist .

FROM debian:stable-slim

ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates \
		netbase \
		&& rm -rf /var/lib/apt/lists/ \
		&& apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/be-classlist_v2/bin /app

WORKDIR /app

EXPOSE 20002

CMD ["./be-classlist"]
