#!/usr/bin/env bash
#
# Builds the given targets twice and compares execution logs to detect
# non-deterministic actions. Exits 0 if deterministic, 1 if not.
#
# Usage:
#   ci/check-determinism.sh //your:targets //other:targets
#   ci/check-determinism.sh  # defaults to //...

set -euo pipefail

TARGETS="${*:-//...}"

bazel build --execution_log_binary_file=build1.log $TARGETS
bazel clean --expunge --async
bazel build --execution_log_binary_file=build2.log $TARGETS

bazel run @bazel_nondeterministic_actions//:check -- \
  --verbose \
  --log_path "$PWD/build1.log" \
  --log_path "$PWD/build2.log"
