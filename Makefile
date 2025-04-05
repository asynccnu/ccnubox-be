SHELL := /bin/sh

# 所有服务名称
SERVICES = be-banner be-calendar be-ccnu be-department be-feed be-static be-user be-website be-elecprice be-grade be-counter be-infosum bff

# 每个服务对应的端口
PORT_be-feed          = 19098
PORT_be-banner        = 19097
PORT_be-calendar      = 19096
PORT_be-department    = 19095
PORT_be-website       = 19094
PORT_be-static        = 19093
PORT_be-ccnu          = 19092
PORT_be-user          = 19091
PORT_be-elecprice     = 19089
PORT_be-grade         = 19088
PORT_be-counter       = 19086
PORT_be-infosum       = 19083
PORT_bff              = 8080

# 镜像标签
TAG ?= latest

# 默认构建所有服务
.PHONY: all $(SERVICES)
all: $(SERVICES)

# 构建指定服务
$(SERVICES):
	@echo "构建服务：$@，端口：$(PORT_$@)，镜像标签：$(TAG)"
	docker build \
		--build-arg Project_Name=$@ \
		--build-arg PORT=$(PORT_$@) \
		-t $@:$(TAG) .

# 通用构建方式：make build Project_Name=be-feed
build:
	@if [ -z "$(Project_Name)" ]; then \
		echo "❌ 请使用 Project_Name=xxx 传入服务名称，例如：make build Project_Name=be-feed"; \
		exit 1; \
	fi

	# 动态获取端口
	@PORT_VAR="PORT_$(Project_Name)"; \
	PORT=$$(eval echo $$${PORT_VAR}); \
	if [ -z "$$PORT" ]; then \
		echo "❌ 未知服务：$(Project_Name)，请检查服务名是否正确"; \
		exit 1; \
	fi; \

	echo "构建服务：$(Project_Name)，端口：$$PORT，镜像标签：$(TAG)"; \
	docker build \
		--build-arg Project_Name=$(Project_Name) \
		--build-arg PORT=$$PORT \
		-t $(Project_Name):$(TAG) .
