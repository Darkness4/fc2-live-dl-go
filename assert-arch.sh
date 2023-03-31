#!/bin/sh

find target/ -type f -executable \( -name '*aarch64*' -o -name '*arm64*' -o -name '*x86_64*' -o -name '*amd64*' \) -print0 | while read -d $'\0' file; do
  expected_arch=$(echo "$file" | grep -o -E 'aarch64|arm64|x86_64|amd64')
  detected_arch=$(file "$file" | grep -o -E 'ARM aarch64|x86-64')
  case "$expected_arch" in
  arm64 | aarch64)
    expected_arch="ARM aarch64"
    ;;
  amd64 | x86_64)
    expected_arch="x86-64"
    ;;
  esac
  if [ "$expected_arch" = "$detected_arch" ]; then
    echo "$file is built for $expected_arch"
  else
    echo "$file does not match expected architecture"
    exit 1
  fi
done
