#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# run statictest
time act --job statictest