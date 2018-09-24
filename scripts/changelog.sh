#!/bin/bash

BRANCH=$1

git-changelog -b $BRANCH -t `git describe --abbrev=0`
