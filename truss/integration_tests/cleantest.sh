#!/bin/bash

# Remove all the dir named `service/` in all dir's in test_service_definitions
for dir in test_service_definitions/*
do
	# Ignore non-directories
	if [ ! -d "$dir" ]; then
		continue
	fi
	echo "$dir"
	rm -rf "$dir"/service
done
