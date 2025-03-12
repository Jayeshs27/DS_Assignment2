PROTO_DIR = protofiles
CLIENT_DIR = client
BANK_SERVER_DIR = server/bank_server
PAYMENT_GATEWAY_DIR = server/payment_gateway
LOGS_DIR = logs

PROTO_FILE = $(PROTO_DIR)/payment_gateway.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

PAYMENT_GATEWAY_FILES := $(wildcard $(PAYMENT_GATEWAY_DIR)/*.go)
BANK_SERVER_FILES := $(wildcard $(BANK_SERVER_DIR)/*.go)
CLIENT_FILES := $(wildcard $(CLIENT_DIR)/*.go)

BANK_ID ?= 1
.PHONY: proto server backend client clean

proto:
	protoc $(GO_FLAGS) $(PROTO_FILE)

payment_gateway:
	go run $(PAYMENT_GATEWAY_FILES) 

bank_server:
	go run $(BANK_SERVER_FILES) $(BANK_ID)

client:
	go run $(CLIENT_FILES)

clean:
	rm -f $(PROTO_OUT_DIR)/*.pb.go

clean_logs:
	rm -f  logs/*/*