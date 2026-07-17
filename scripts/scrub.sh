#!/bin/sh
# Scrub the lfscache object store: delete any cached object whose sha256 does
# not match its oid filename. Run against the cache directory (the directory
# passed to lfscache -directory), e.g.:
#
#   ./scrub.sh /var/cache/lfscache
#
# Safe to run while the server is live: a deleted entry is refetched on next
# request. Corrupt temp files are cleaned up by the server itself.
set -eu

dir="${1:?usage: scrub.sh <cache-directory>}/objects"
scanned=0
deleted=0

for f in $(find "$dir" -type f); do
	scanned=$((scanned + 1))
	oid=$(basename "$f")
	actual=$(sha256sum "$f" | cut -d' ' -f1)
	if [ "$oid" != "$actual" ]; then
		deleted=$((deleted + 1))
		echo "corrupt: $oid (sha256 $actual)"
		rm -f "$f"
	fi
done

echo "scanned $scanned objects, deleted $deleted corrupt entries"
