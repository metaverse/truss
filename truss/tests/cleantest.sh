#!/bin/bash

# Remove all the dir named `service/` in all dir's in $1
# COULD BE DANGEROUS
for dir in "$1"*
do
	echo "$dir"
	rm -r "$dir"/service 2>/dev/null
done
