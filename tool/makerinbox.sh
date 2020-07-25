#!/bin/sh

url="$1"
local_artwork=/tmp/sgtm-track-$(date +%s).jpg
artwork=$(curl "$url?format=json" | jq -r .artworkUrl | sed s/-large.jpg/-t500x500.jpg/)
wget -q -O "$local_artwork" "$artwork"
ls -la $local_artwork
echo ""
(
    set -x
    makerinbox done --attach "$local_artwork" "🎶 daily music production $url #ultreme"
)
echo ""
