#! /bin/bash

function gentestdata {
	protoc -I/usr/local/include -I. -I/home/adamryman/projects/go/src/github.com/TuneLab/gob/testproto/third_party/googleapis --truss-protocast_out=./data/data $1
}

if ! which protoc-gen-truss-protocast > /dev/null; then
	echo "protoc-gen-truss-protocast not in \$PATH, installing..."
	go install github.com/TuneLab/gob/protoc-gen-truss-protocast
fi

mkdir -p ./data/data

for file in ./definitions/*
do
	# Ignore directories
	if [ -d $file ]; then
		continue
	fi
	echo "Generating protoc output for "$(basename $file)"..."
	gentestdata "$file"
done

