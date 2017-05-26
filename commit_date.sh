#!/usr/bin/env bash
# commit_date.sh outputs the date of the last commit in UTC
# Works with both gnu date and bsd date

set -e
[[ -z $DEBUG ]] || set -x

# get the epoc time of the commit
head_commit=$(git rev-parse HEAD)
git_commit_epoc="$(git show -s --format=%ct $head_commit)"

# use date for fomatting
# bsd date does not have `--version`
if [[ "$(date --version 2>/dev/null 1>/dev/null; echo $?)" -eq "1" ]]; then
	# bsd date
	commit_date=$(date -r $git_commit_epoc -u)
else
	# gnu date
	commit_date=$(date --date="@$git_commit_epoc" -u)
fi

echo $commit_date
