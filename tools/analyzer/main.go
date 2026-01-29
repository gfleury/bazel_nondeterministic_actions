package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type lineType int

const (
	lineActionSeparator lineType = iota
	lineSection
	lineMnemonic
	lineRemotable
	lineCacheable
	lineDiff
	lineBoring
)

type parsedLine struct {
	typ           lineType
	stringPayload string
	boolPayload   bool
}

var (
	diffRe    = regexp.MustCompile(`^[^ ]`)
	sectionRe = regexp.MustCompile(`^  ([^ ]+) \{`)
	kvRe      = regexp.MustCompile(`^  ([^ ]+): (.+)`)
)

const separator = "  ---------------------------------------------------------"

func parseLine(text string) parsedLine {
	if text == separator {
		return parsedLine{typ: lineActionSeparator}
	}

	if diffRe.MatchString(text) {
		return parsedLine{typ: lineDiff}
	}

	if m := sectionRe.FindStringSubmatch(text); m != nil {
		return parsedLine{typ: lineSection, stringPayload: m[1]}
	}

	if m := kvRe.FindStringSubmatch(text); m != nil {
		switch m[1] {
		case "mnemonic":
			return parsedLine{typ: lineMnemonic, stringPayload: m[2]}
		case "remotable":
			b, err := strconv.ParseBool(m[2])
			if err != nil {
				panic(fmt.Sprintf("failed to parse remotable value %q: %v", m[2], err))
			}
			return parsedLine{typ: lineRemotable, boolPayload: b}
		case "cacheable":
			b, err := strconv.ParseBool(m[2])
			if err != nil {
				panic(fmt.Sprintf("failed to parse cacheable value %q: %v", m[2], err))
			}
			return parsedLine{typ: lineCacheable, boolPayload: b}
		}
	}

	return parsedLine{typ: lineBoring}
}

func printSummary(actionCount int64, lineCount int, started time.Time) {
	elapsed := time.Since(started).Seconds()
	fmt.Printf("Processed %5d total messages in %.1f seconds (%7.0f messages/sec, %10.0f lines/sec)\n",
		actionCount, elapsed, float64(actionCount)/elapsed, float64(lineCount)/elapsed)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: analyzer <diff-file>")
		os.Exit(1)
	}

	started := time.Now()
	diffPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Open(diffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var curr []string
	var actionCount int64
	var lineCount int

	var remotable bool
	var cacheable bool
	var section string
	var hasSection bool
	var sectionDiffs []string
	var mnemonic string
	_ = mnemonic // mnemonic tracked but not used for output (matches Rust behavior)

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		pl := parseLine(line)

		switch pl.typ {
		case lineActionSeparator:
			actionCount++
			if actionCount%1000 == 0 {
				printSummary(actionCount, lineCount-1, started)
			}

			if len(sectionDiffs) > 0 && (remotable || cacheable) {
				fmt.Println(strings.Join(curr, "\n"))
			}

			curr = curr[:0]
			remotable = false
			cacheable = false
			hasSection = false
			section = ""
			sectionDiffs = sectionDiffs[:0]
			mnemonic = ""

		case lineSection:
			hasSection = true
			section = pl.stringPayload

		case lineRemotable:
			remotable = pl.boolPayload

		case lineCacheable:
			cacheable = pl.boolPayload

		case lineMnemonic:
			mnemonic = pl.stringPayload

		case lineDiff:
			if hasSection {
				sectionDiffs = append(sectionDiffs, section)
			} else if len(curr) > 0 {
				panic(fmt.Sprintf("%q:%d: Diff outside of a section!", diffPath, lineCount))
			}

		case lineBoring:
			// nothing
		}

		curr = append(curr, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	printSummary(actionCount, lineCount, started)
}
