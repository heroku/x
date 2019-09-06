#!/bin/bash

if [ ! -z "$(git status --porcelain)" ]; then
	echo "!!! Uncommitted changes, can't continue unless git state is clean. Commit / stash and try again"
	echo
	git status
	exit 1
fi

go mod tidy
go mod vendor

if [ ! -z "$(git status --porcelain)" ]; then
	echo "!!! 'go mod tidy; go mod vendor' introduced changes"
	echo "!!! run those commands locally and commit the changes"
	echo
	git status
	exit 1
fi