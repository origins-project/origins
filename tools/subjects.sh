#!/usr/bin/env bash
set -euo pipefail

# Reads commit subjects or pull request titles, one a line, and fails on any that is
# not `type(context): summary` within 72 columns, the form the release notes are made
# from. The ` (#N)` GitHub appends to a squashed pull request is not counted.
#
#   git log --format=%s origin/master..HEAD | tools/subjects.sh
form='^(feat|fix|docs|refactor|test|build|chore)\([a-z0-9-]+\)!?: [^ ]'
bad=0
while IFS= read -r subject || [ -n "$subject" ]; do
	bare="${subject% (#[0-9]*)}"
	if ! [[ "$bare" =~ $form ]] || [ "${#bare}" -gt 72 ]; then
		echo "not type(context): summary within 72 columns: $subject" >&2
		bad=1
	fi
done
exit "$bad"
