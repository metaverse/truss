#!/bin/bash

# Go through dir's in $1, go into them, run truss on them, cd to pervious dir for the next
for dir in "$1"*
do
	echo "$dir"
	cd "$dir" && \
		truss *.proto &&\
		echo $PWD && \
		cd - && \
		echo $PWD
done
