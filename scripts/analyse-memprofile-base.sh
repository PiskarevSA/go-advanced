#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# analyse memprofile
go tool pprof -http=":9090" usecases-base.test profiles/base.pprof