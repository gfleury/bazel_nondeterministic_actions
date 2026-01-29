#!/usr/bin/env bash

set -euxo pipefail

echo "$(date) $RANDOM" >example/example.txt
./tools/bazel build --execution_log_binary_file=build1.log example:uses_example_file.txt
echo "$(date) $RANDOM" >example/example.txt
./tools/bazel build --execution_log_binary_file=build2.log example:uses_example_file.txt

./tools/bazel run //:check -- --log_path "$PWD/build1.log" --log_path "$PWD/build2.log"
