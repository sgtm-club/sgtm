#!/bin/sh -e

URL=${1:-https://sgtm.club}
set -x
http $URL/api/v1/PostList | jq -r '.posts[].ipfs_cid | select(.!=null)' | xargs -t ipfs pin add
