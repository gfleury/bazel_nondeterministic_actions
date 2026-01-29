package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	execlog "tools/execlog/lib"
	pb "tools/execlog/proto"
	"google.golang.org/protobuf/proto"
)

type stringSlice []string

func (s *stringSlice) String() string { return fmt.Sprintf("%v", *s) }
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Exit codes.
const (
	exitDeterministic    = 0
	exitNonDeterministic = 1
	exitUsageError       = 2
)

// diffSections compares two SpawnExec protos field-group by field-group and
// returns a list of section names that differ.
func diffSections(a, b *pb.SpawnExec) []string {
	var diffs []string

	// command_args
	if !proto.Equal(
		&pb.SpawnExec{CommandArgs: a.CommandArgs},
		&pb.SpawnExec{CommandArgs: b.CommandArgs},
	) {
		diffs = append(diffs, "command_args")
	}

	// environment_variables
	if !proto.Equal(
		&pb.SpawnExec{EnvironmentVariables: a.EnvironmentVariables},
		&pb.SpawnExec{EnvironmentVariables: b.EnvironmentVariables},
	) {
		diffs = append(diffs, "environment_variables")
	}

	// platform
	if !proto.Equal(
		&pb.SpawnExec{Platform: a.Platform},
		&pb.SpawnExec{Platform: b.Platform},
	) {
		diffs = append(diffs, "platform")
	}

	// inputs
	if !proto.Equal(
		&pb.SpawnExec{Inputs: a.Inputs},
		&pb.SpawnExec{Inputs: b.Inputs},
	) {
		diffs = append(diffs, "inputs")
	}

	// listed_outputs
	if !proto.Equal(
		&pb.SpawnExec{ListedOutputs: a.ListedOutputs},
		&pb.SpawnExec{ListedOutputs: b.ListedOutputs},
	) {
		diffs = append(diffs, "listed_outputs")
	}

	// actual_outputs
	if !proto.Equal(
		&pb.SpawnExec{ActualOutputs: a.ActualOutputs},
		&pb.SpawnExec{ActualOutputs: b.ActualOutputs},
	) {
		diffs = append(diffs, "actual_outputs")
	}

	return diffs
}

// actionKey returns the pairing key for a SpawnExec (first listed output).
func actionKey(exec *pb.SpawnExec) string {
	return execlog.GetFirstOutput(exec)
}

// run is the testable entry point. It returns an exit code.
func run(paths []string, runner string) int {
	if len(paths) != 2 {
		fmt.Fprintf(os.Stderr, "Error: exactly two --log_path values required, got %d\n", len(paths))
		return exitUsageError
	}

	// Phase 1: Parse log1 → collect all SpawnExec, build Golden.
	f1, err := os.Open(paths[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", paths[0], err)
		return exitUsageError
	}
	defer f1.Close()

	parser1 := execlog.NewFilteringParser(f1, runner)
	golden := execlog.NewGolden()
	log1Actions := make(map[string]*pb.SpawnExec)

	for {
		exec, err := parser1.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", paths[0], err)
			return exitUsageError
		}
		if exec == nil {
			break
		}
		golden.AddSpawnExec(exec)
		key := actionKey(exec)
		if key != "" {
			log1Actions[key] = exec
		}
	}

	// Phase 2: Parse log2 with reordering.
	f2, err := os.Open(paths[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", paths[1], err)
		return exitUsageError
	}
	defer f2.Close()

	parser2 := execlog.NewFilteringParser(f2, runner)
	reordered, err := execlog.NewReorderingParser(golden, parser2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", paths[1], err)
		return exitUsageError
	}

	log2Actions := make(map[string]*pb.SpawnExec)
	for {
		exec, err := reordered.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading reordered %s: %v\n", paths[1], err)
			return exitUsageError
		}
		if exec == nil {
			break
		}
		key := actionKey(exec)
		if key != "" {
			log2Actions[key] = exec
		}
	}

	// Phase 3: Compare paired actions.
	type diffResult struct {
		key      string
		mnemonic string
		sections []string
	}

	var nonDeterministic []diffResult
	var skippedCount int
	var uniqueToLog1 []string
	var uniqueToLog2 []string

	for key, a := range log1Actions {
		b, ok := log2Actions[key]
		if !ok {
			uniqueToLog1 = append(uniqueToLog1, key)
			continue
		}

		// Fast path: proto.Equal skips detailed comparison.
		if proto.Equal(a, b) {
			continue
		}

		// Only report non-determinism for remotable or cacheable actions.
		if !a.Remotable && !a.Cacheable {
			skippedCount++
			continue
		}

		sections := diffSections(a, b)
		if len(sections) > 0 {
			mnemonic := a.Mnemonic
			if mnemonic == "" {
				mnemonic = "(unknown)"
			}
			nonDeterministic = append(nonDeterministic, diffResult{
				key:      key,
				mnemonic: mnemonic,
				sections: sections,
			})
		}
	}

	for key := range log2Actions {
		if _, ok := log1Actions[key]; !ok {
			uniqueToLog2 = append(uniqueToLog2, key)
		}
	}

	// Phase 4: Print report.
	if len(nonDeterministic) > 0 {
		fmt.Printf("Non-deterministic actions found: %d\n\n", len(nonDeterministic))
		for _, d := range nonDeterministic {
			fmt.Printf("  %s [%s]\n", d.key, d.mnemonic)
			fmt.Printf("    differs in: %s\n", strings.Join(d.sections, ", "))
		}
		fmt.Println()
	}

	if skippedCount > 0 {
		fmt.Printf("Skipped %d non-remotable/non-cacheable differing action(s)\n", skippedCount)
	}

	if len(uniqueToLog1) > 0 {
		fmt.Printf("Actions unique to log1: %d\n", len(uniqueToLog1))
		for _, k := range uniqueToLog1 {
			fmt.Printf("  %s\n", k)
		}
	}

	if len(uniqueToLog2) > 0 {
		fmt.Printf("Actions unique to log2: %d\n", len(uniqueToLog2))
		for _, k := range uniqueToLog2 {
			fmt.Printf("  %s\n", k)
		}
	}

	// Summary line.
	totalPaired := len(log1Actions) - len(uniqueToLog1)
	fmt.Printf("\nSummary: %d paired actions compared, %d non-deterministic\n",
		totalPaired, len(nonDeterministic))

	if len(nonDeterministic) > 0 {
		return exitNonDeterministic
	}
	return exitDeterministic
}

func main() {
	var logPaths stringSlice
	var runner string
	flag.Var(&logPaths, "log_path", "Input binary protobuf log file (must be specified exactly twice)")
	flag.StringVar(&runner, "restrict_to_runner", "", "Filter to specific runner")
	flag.Parse()

	os.Exit(run(logPaths, runner))
}
