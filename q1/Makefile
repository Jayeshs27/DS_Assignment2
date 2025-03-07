PROTO_DIR = protofiles
CLIENT_DIR = client
LB_SERVER_DIR = server/load_balancer
BACKEND_SERVER_DIR = server/backend_server

PROTO_FILE_GREET = $(PROTO_DIR)/load_balancing.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

.PHONY: proto server backend client clean

POLICY ?= PF
TASK ?= 0

proto:
	protoc $(GO_FLAGS) $(PROTO_FILE_GREET)

server:
	go run $(LB_SERVER_DIR)/main.go $(POLICY)

backend:
	go run $(BACKEND_SERVER_DIR)/main.go $(BACKEND_SERVER_DIR)/tasks.go

client:
	go run $(CLIENT_DIR)/main.go $(TASK)

clean:
	rm -f $(PROTO_OUT_DIR)/*.pb.go
