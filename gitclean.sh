#!/bin/bash

git filter-branch --force --index-filter "git rm -rf --cached --ignore-unmatch $1" --prune-empty --tag-name-filter cat -- --all
git for-each-ref --format='delete %(refname)' refs/original | git update-ref --stdin
git reflog expire --expire=now --all
git gc --prune=now

git rev-list --objects --all | grep "$(git verify-pack -v .git/objects/pack/*.idx | sort -k 3 -n | tail -3 | awk '{print$1}')"
