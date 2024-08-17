#!/bin/bash

set -euo pipefail

if [ ! -f "test.mp4" ]; then
  echo "Please provide a test.mp4 file"
  exit 1
fi

mkdir -p ./test-output
rm -f ./test-output/*

# Initial test
"${EXECUTABLE}" --help

# Now actually run
"${EXECUTABLE}" --debug watch -c config.yaml.test &
PID=$!
clean() {
  kill -15 $PID
}

trap clean EXIT

sleep 10

# Stream video to FC2
docker run --rm \
  -v "$(pwd):/in" \
  linuxserver/ffmpeg:7.0.1 \
  -re -i /in/test.mp4 \
  -c:v libx264 \
  -preset veryfast \
  -b:v 3000k \
  -maxrate 3000k \
  -bufsize 6000k \
  -pix_fmt yuv420p \
  -g 50 \
  -c:a aac \
  -b:a 160k \
  -ac 2 \
  -ar 44100 \
  -f flv \
  "$RTMP_URL"

sleep 10
