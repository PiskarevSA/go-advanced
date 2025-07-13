#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# generate rsa keys
# .. private key (for server)
openssl genpkey -algorithm RSA -out cmd/server/private.pem -pkeyopt rsa_keygen_bits:4096
# .. public key (for agent)
openssl rsa -pubout -in cmd/server/private.pem -out cmd/agent/public.pem