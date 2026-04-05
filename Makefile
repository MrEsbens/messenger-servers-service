.PHONY: gen-proto install-tools

# Путь к папке с proto файлами
PROTO_DIR := proto
# Путь куда генерировать (соответствует go_package в .proto)
API_DIR := api/servers/v1

gen-proto:
	@echo "Generating gRPC code..."
	@mkdir -p $(API_DIR)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(API_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(API_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/servers.proto