#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# run memprofile
go test github.com/PiskarevSA/go-advanced/internal/usecases -bench=. -memprofile=profiles/base.pprof
mv usecases.test usecases-base.test