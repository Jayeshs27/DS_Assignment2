PROTO_DIR = protofiles
CLIENT_DIR = client
BANK_SERVER_DIR = server/bank_server
PAYMENT_GATEWAY_DIR = server/payment_gateway
LOGS_DIR = logs

PROTO_FILE_GREET = $(PROTO_DIR)/payment_gateway.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

.PHONY: proto server backend client clean

proto:
	protoc $(GO_FLAGS) $(PROTO_FILE_GREET)

payment_gateway:
	go run $(PAYMENT_GATEWAY_DIR)/main.go $(PAYMENT_GATEWAY_DIR)/interceptor.go

bank_server:
	go run $(BANK_SERVER_DIR)/main.go 

client:
	go run $(CLIENT_DIR)/main.go $(CLIENT_DIR)/logging.go

clean:
	rm -f $(PROTO_OUT_DIR)/*.pb.go

clean_logs:
	rm -f  logs/*/*