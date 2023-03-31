#!/bin/sh

find ./target/*/* -type f -exec sh -c 'f="$1"; echo "mv -i "$f" "$(dirname "$f")/.."";' shell {} \; | less

echo "Confirm move (y/n)? "
read -r answer
if [ "$answer" != "${answer#[Yy]}" ]; then
  find ./target/*/* -type f -exec sh -c 'f="$1"; mv -i "$f" "$(dirname "$f")/..";' shell {} \;
else
  echo "Move canceled."
fi
