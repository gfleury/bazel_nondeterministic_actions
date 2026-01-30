# bazel_nondeterministic_actions

A Bazel module that detects non-deterministic build actions by comparing two execution logs.

## How it works

Bazel can emit a binary execution log (`--experimental_execution_log_file`) containing every action it ran. This tool parses two such logs, pairs actions by their primary output, and reports which remotable/cacheable actions produced different results between the two builds.

Exit codes:
- `0` — deterministic (all paired actions match)
- `1` — non-determinism found
- `2` — usage error

## Usage from another repository

Add this module as a dependency in your `MODULE.bazel`:

### Git repository

```starlark
bazel_dep(name = "bazel_nondeterministic_actions", version = "0.0.0")

git_override(
    module_name = "bazel_nondeterministic_actions",
    remote = "https://github.com/yourorg/bazel_nondeterministic_actions.git",
    commit = "COMMIT_SHA",
)
```

### Local path (for development)

```starlark
bazel_dep(name = "bazel_nondeterministic_actions", version = "0.0.0")

local_path_override(
    module_name = "bazel_nondeterministic_actions",
    path = "/absolute/path/to/bazel_nondeterministic_actions",
)
```

### Bazel Central Registry

If published to BCR, just the `bazel_dep` line is needed:

```starlark
bazel_dep(name = "bazel_nondeterministic_actions", version = "0.0.0")
```

### Running the determinism check

The simplest way is to use the bundled `check-determinism` script, which
builds your targets twice, cleans between runs, and compares the execution logs
automatically:

```bash
bazel run @bazel_nondeterministic_actions//:check-determinism -- //your:targets
```

If no targets are given it defaults to `//...`.

### Running the check tool manually

You can also generate two execution logs yourself and run the check tool directly:

```bash
bazel build --execution_log_binary_file=build1.log //your:target
bazel build --execution_log_binary_file=build2.log //your:target

bazel run @bazel_nondeterministic_actions//:check -- \
  --log_path /abs/path/build1.log \
  --log_path /abs/path/build2.log
```

Log paths must be absolute since `bazel run` executes from a runfiles directory.

### Flags

| Flag | Description |
|------|-------------|
| `--log_path` | Path to a binary execution log (specify exactly twice) |
| `--restrict_to_runner` | Only compare actions with this runner (e.g. `linux-sandbox`) |

## Usage within this repository

Run the full determinism check:

```bash
./tools/bazel run //:check-determinism -- //your:targets
```

Or run the check tool directly against existing logs:

```bash
./tools/bazel run //:check -- --log_path "$PWD/build1.log" --log_path "$PWD/build2.log"
```

Or run the included example end-to-end:

```bash
./example.sh
```

## Running tests

```bash
./tools/bazel test //tools/...
```
