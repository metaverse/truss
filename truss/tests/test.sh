#!/bin/bash
red='\e[0;31m'
yellow='\e[1;33m'
green='\e[0;32m'
reset='\e[0m'

STATUS=0

ORIGIN_DIR=$PWD

# Go through dir's in $1, go into them, run truss on them, cd to pervious dir for the next
for dir in "$1"*
do
	# Ignore non-directories
	if [ ! -d "$dir" ]; then
		continue
	fi
	echo -en "$yellow""Running integration test $dir ... "

	cd "$dir"
	output="$(truss *.proto 2>&1)"
	if [ $? -eq 0 ]; then
		echo -e "$green""SUCCESS: $dir passed integration test$reset"
	else
		echo -e "$red""ERROR: $dir failed integration test!$reset"
		echo -e "$output"
		STATUS=1
	fi
	cd "$ORIGIN_DIR"
done

exit $STATUS
