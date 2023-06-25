#!/bin/bash
set -e -x

rm -rf coverdata/ && mkdir coverdata
go build -cover .
./x
go tool covdata textfmt -i=coverdata -o profile.txt
