#!/usr/bin/env bash
#
# Builds the given targets twice and compares execution logs to detect
# non-deterministic actions. Exits 0 if deterministic, 1 if not.
#
# Usage:
#   bazel run @bazel_nondeterministic_actions//:check-determinism -- //your:targets
#   bazel run //:check-determinism  # defaults to //...

set -euo pipefail

# Locate the check binary from runfiles.
CHECK_BIN="$(dirname "$0")/../tools/check/check_/check"
if [[ ! -x "$CHECK_BIN" ]]; then
  # Fallback: try the runfiles directory layout
  CHECK_BIN="$0.runfiles/bazel_nondeterministic_actions/tools/check/check_/check"
fi

# BUILD_WORKSPACE_DIRECTORY is set by `bazel run` — it points to the
# consumer's workspace root so bazel build commands execute there.
cd "${BUILD_WORKSPACE_DIRECTORY:-.}"

TARGETS="${*:-//...}"

bazel build --execution_log_binary_file=build1.log $TARGETS
bazel clean --expunge --async
bazel build --execution_log_binary_file=build2.log $TARGETS

exec "$CHECK_BIN" \
  --verbose \
  --log_path "$PWD/build1.log" \
  --log_path "$PWD/build2.log"
