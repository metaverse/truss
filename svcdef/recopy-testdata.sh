#!/bin/bash

mkdir scrap/ 
cd scrap/ 
cp ../test-proto.txt ./svc.proto
truss *.proto
cp TEST-service/svc.pb.go ../test-go.txt
cd ../
rm -r scrap
