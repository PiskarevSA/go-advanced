#!/usr/bin/bash
# go to root directory
SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/..
# update .github/workflows directory from main branch of the repo
# https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
git fetch template && git checkout template/main .github