#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# run memprofile
go test github.com/PiskarevSA/go-advanced/internal/usecases -bench=. -memprofile=profiles/result.pprof
mv usecases.test usecases-result.test