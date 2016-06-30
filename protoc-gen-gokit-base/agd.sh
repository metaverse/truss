#!/bin/bash

# agd.sh searches for file $1 in generate and to_generate dir's and diffs them with less and color output


# This function allows to to compare two files using git's diff config
# These files do not need to be in git repos
# You can pipe this into anything and have the output be nice
function diffc {
	git --no-pager diff --color=always --no-index $1 $2
}

# This function searches for two files in generate and to_generate and diffs them
# See if there are differences between generated service and orginal
function agd {
	files=$(ag --ignore template_files --ignore protoc-gen-gokit-base -g generate -g to-generate -g $1)
	IFS=$'\n'
	lines=($files)

	# ${#array[@]} gives you the length of a bash array
	if [ ${#lines[@]} == 0 ]; then
		echo "No files found"
		exit
	fi

	if [ ${#lines[@]} == 2 ]; then
		original=1
		generated=0
	else
		count=0
		ag --ignore template_files --ignore protoc-gen-gokit-base -g generate -g to-generate -g $1| while read line; do
			echo $count $line
			(( count++ ))
		done

		echo "Select original file:"
		read original
		echo "Select generated file:"
		read generated
	fi

	echo "diffing ${lines[$original]} and ${lines[$generated]}"
	diffc ${lines[$original]} ${lines[$generated]} | less -XFR
}

agd $1
