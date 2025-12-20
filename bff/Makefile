DOCS_DIR := docs
SWAGGER_FILE := $(DOCS_DIR)/swagger.yaml
OPENAPI_FILE := $(DOCS_DIR)/openapi3.yaml

.PHONY: swag
swag:
	@echo === 生成 Swagger 文档中 ===
	@swag init
	@echo === Swagger 文档已生成: $(SWAGGER_FILE) ===
	@echo === 转换为 OpenAPI3... ===
	@swagger2openapi $(SWAGGER_FILE) -o $(OPENAPI_FILE) --patch
	@echo === OpenAPI 3 文档已生成: $(OPENAPI_FILE) ===
