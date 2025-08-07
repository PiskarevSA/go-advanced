#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# generate proto
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  internal/proto/metrics.proto