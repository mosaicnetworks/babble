#!/bin/bash

if [[ -d "./.git/hooks" ]]
then
  cp ./scripts/commit-msg ./.git/hooks

  chmod +x ./.git/hooks/commit-msg
else
  echo "Not in a repo or not launched from repo root."
fi
