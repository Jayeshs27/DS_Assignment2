PROTO_DIR = protofiles
MASTER_DIR = master

PROTO_FILE_MAP_REDUCE = $(PROTO_DIR)/mapreduce.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

.PHONY: proto master clean

MASTER_DIR_FILES := $(wildcard $(MASTER_DIR)/*.go)

DATA_DIR ?= dataset
NUM_REDUCE ?= 3

proto:
	protoc $(GO_FLAGS) $(PROTO_FILE_MAP_REDUCE)

master:
	go run $(MASTER_DIR_FILES) $(DATA_DIR) $(NUM_REDUCE)

clean:
	rm -f $(PROTO_OUT_DIR)/*.pb.go
