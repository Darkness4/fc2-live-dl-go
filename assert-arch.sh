#!/bin/sh

find target/ -type f -executable \( -name '*aarch64*' -o -name '*arm64*' -o -name '*x86_64*' -o -name '*amd64*' -o -name '*x86-64*' -o -name '*riscv64*' \) -print0 | while read -d $'\0' file; do
  expected_arch=$(echo "$file" | cut -d ':' -f 2- | grep -o -E 'aarch64|arm64|x86_64|x86-64|amd64|RISC-V|riscv64')
  detected_arch=$(file "$file" | cut -d ':' -f 2- | grep -o -E 'aarch64|arm64|x86_64|x86-64|amd64|RISC-V|riscv64')
  case "$expected_arch" in
  arm64 | aarch64)
    expected_arch="arm64"
    ;;
  amd64 | x86_64 | x86-64)
    expected_arch="amd64"
    ;;
  RISC-V | riscv64)
    detected_arch="riscv64"
    ;;
  esac
  case "$detected_arch" in
  arm64 | aarch64)
    detected_arch="arm64"
    ;;
  amd64 | x86_64 | x86-64)
    detected_arch="amd64"
    ;;
  RISC-V | riscv64)
    detected_arch="riscv64"
    ;;
  esac
  if [ "$expected_arch" = "$detected_arch" ]; then
    echo "$file is built for $expected_arch"
  else
    echo "$file does not match expected architecture"
    exit 1
  fi
done
