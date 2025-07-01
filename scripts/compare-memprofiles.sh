#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# compare memprofiles
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof > profiles/diff.txt