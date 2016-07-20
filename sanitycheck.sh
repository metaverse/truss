#!/bin/bash

set -x #echo on

# If any of these error we might have an issue 
# Until we have a better development workflow then pushing to master
go get github.com/TuneLab/gob/...
cd protoc-gen-truss-gokit && \
make && \
make clean && \
cd -
