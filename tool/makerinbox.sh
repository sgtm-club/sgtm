#!/bin/sh

url="$1"

rm -f /tmp/sgtm-track.jpg
artwork=$(curl "$url?format=json" | jq -r .artworkUrl | sed s/-large.jpg/-t500x500.jpg/)
wget -q  -O /tmp/sgtm-track.jpg "$artwork"
ls -la /tmp/sgtm-track.jpg
set -x
makerinbox done --attach=/tmp/sgtm-track.jpg "🎶 daily music production $url #ultreme"

