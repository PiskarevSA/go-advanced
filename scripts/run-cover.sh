#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# run cover
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out