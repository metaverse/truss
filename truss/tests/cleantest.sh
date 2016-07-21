#!/bin/bash

# Remove all the dir named `service/` in all dir's in $1
# COULD BE DANGEROUS
for dir in "$1"*
do
	# Ignore non-directories
	if [ ! -d "$dir" ]; then
		continue
	fi
	echo "$dir"
	rm -rf "$dir"/service
done
