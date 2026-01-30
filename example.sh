#!/usr/bin/env bash

set -euxo pipefail

TARGETS="example:uses_example_file.txt example:leaks_user example:leaks_hostname example:leaks_path example:leaks_multiple_env example:leaks_env_in_config"

# Build 1: with one set of env vars passed to actions
echo "$(date) $RANDOM" >example/example.txt
./tools/bazel build --execution_log_binary_file=build1.log \
  --action_env=USER=alice \
  --action_env=HOSTNAME=build-server-1 \
  --action_env=HOME=/home/alice \
  --action_env=LANG=en_US.UTF-8 \
  $TARGETS

./tools/bazel clean --expunge --async

# Build 2: with different env vars (simulates a different CI runner)
echo "$(date) $RANDOM" >example/example.txt
./tools/bazel build --execution_log_binary_file=build2.log \
  --action_env=USER=bob \
  --action_env=HOSTNAME=build-server-2 \
  --action_env=HOME=/home/bob \
  --action_env=LANG=C.UTF-8 \
  $TARGETS

./tools/bazel run //:check -- --verbose --log_path "$PWD/build1.log" --log_path "$PWD/build2.log"
