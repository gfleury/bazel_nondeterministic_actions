package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	execlog "tools/execlog/lib"
)

type stringSlice []string

func (s *stringSlice) String() string { return fmt.Sprintf("%v", *s) }
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	logPaths         stringSlice
	outputPaths      stringSlice
	restrictToRunner = flag.String("restrict_to_runner", "", "Filter to specific runner")
)

func init() {
	flag.Var(&logPaths, "log_path", "Input binary protobuf log file (can be specified 1-2 times)")
	flag.Var(&outputPaths, "output_path", "Output text file (can be specified 0-2 times)")
}

const delimiter = "\n---------------------------------------------------------\n"

func output(p execlog.Parser, w io.Writer, golden *execlog.Golden) error {
	for {
		exec, err := p.Next()
		if err != nil {
			return err
		}
		if exec == nil {
			break
		}
		if err := execlog.FormatSpawnExec(w, exec); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, delimiter); err != nil {
			return err
		}
		if golden != nil {
			golden.AddSpawnExec(exec)
		}
	}
	return nil
}

func processFile(logPath, outputPath, runner string, golden *execlog.Golden) error {
	f, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	parser := execlog.NewFilteringParser(f, runner)

	var w io.Writer
	if outputPath == "" {
		w = os.Stdout
	} else {
		outFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		w = outFile
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	return output(parser, bw, golden)
}

func processSecondFile(logPath, outputPath, runner string, golden *execlog.Golden) error {
	f, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	parser := execlog.NewFilteringParser(f, runner)
	reorderingParser, err := execlog.NewReorderingParser(golden, parser)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	bw := bufio.NewWriter(outFile)
	defer bw.Flush()

	return output(reorderingParser, bw, nil)
}

func main() {
	flag.Parse()

	if len(logPaths) == 0 {
		fmt.Fprintln(os.Stderr, "--log_path needs to be specified.")
		os.Exit(1)
	}
	if len(outputPaths) > len(logPaths) {
		fmt.Fprintln(os.Stderr, "Too many --output_path values.")
		os.Exit(1)
	}
	if len(logPaths) > 2 {
		fmt.Fprintln(os.Stderr, "Too many --log_path: at most two files are currently supported.")
		os.Exit(1)
	}
	if len(logPaths) == 2 && len(outputPaths) != 2 {
		fmt.Fprintln(os.Stderr, "Exactly two --output_path values expected, one for each of --log_path values.")
		os.Exit(1)
	}

	logPath := logPaths[0]
	var secondPath string
	var output1Path, output2Path string

	if len(logPaths) > 1 {
		secondPath = logPaths[1]
		output1Path = outputPaths[0]
		output2Path = outputPaths[1]
	} else if len(outputPaths) > 0 {
		output1Path = outputPaths[0]
	}

	var golden *execlog.Golden
	if secondPath != "" {
		golden = execlog.NewGolden()
	}

	if err := processFile(logPath, output1Path, *restrictToRunner, golden); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", logPath, err)
		os.Exit(1)
	}

	if secondPath != "" {
		if err := processSecondFile(secondPath, output2Path, *restrictToRunner, golden); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", secondPath, err)
			os.Exit(1)
		}
	}
}
