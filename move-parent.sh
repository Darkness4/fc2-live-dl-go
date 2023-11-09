#!/bin/sh

set -ex

find ./target/*/* -type f -exec sh -c 'f="$1"; mv -i "$f" "$(dirname "$f")/..";' shell {} \;
