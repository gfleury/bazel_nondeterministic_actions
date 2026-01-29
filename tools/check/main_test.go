package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	pb "tools/execlog/proto"
	"google.golang.org/protobuf/encoding/protodelim"
)

// writeLogs writes SpawnExec records to a temporary file and returns the path.
func writeLogs(t *testing.T, dir, name string, execs []*pb.SpawnExec) string {
	t.Helper()
	path := filepath.Join(dir, name)
	var buf bytes.Buffer
	for _, exec := range execs {
		if _, err := protodelim.MarshalTo(&buf, exec); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIdenticalLogs_Exit0(t *testing.T) {
	dir := t.TempDir()
	actions := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "hello"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc123", SizeBytes: 10}},
			},
		},
	}
	log1 := writeLogs(t, dir, "log1.bin", actions)
	log2 := writeLogs(t, dir, "log2.bin", actions)

	code := run([]string{log1, log2}, "")
	if code != exitDeterministic {
		t.Errorf("identical logs: got exit code %d, want %d", code, exitDeterministic)
	}
}

func TestDifferentActualOutputs_Remotable_Exit1(t *testing.T) {
	dir := t.TempDir()
	actions1 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "hello"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc123", SizeBytes: 10}},
			},
		},
	}
	actions2 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "hello"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "def456", SizeBytes: 10}},
			},
		},
	}
	log1 := writeLogs(t, dir, "log1.bin", actions1)
	log2 := writeLogs(t, dir, "log2.bin", actions2)

	code := run([]string{log1, log2}, "")
	if code != exitNonDeterministic {
		t.Errorf("different actual_outputs on remotable action: got exit code %d, want %d", code, exitNonDeterministic)
	}
}

func TestDifferentOutputs_NotRemotableNotCacheable_Exit0(t *testing.T) {
	dir := t.TempDir()
	actions1 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "hello"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     false,
			Cacheable:     false,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc123", SizeBytes: 10}},
			},
		},
	}
	actions2 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "hello"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     false,
			Cacheable:     false,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "def456", SizeBytes: 10}},
			},
		},
	}
	log1 := writeLogs(t, dir, "log1.bin", actions1)
	log2 := writeLogs(t, dir, "log2.bin", actions2)

	code := run([]string{log1, log2}, "")
	if code != exitDeterministic {
		t.Errorf("non-remotable/non-cacheable diff: got exit code %d, want %d", code, exitDeterministic)
	}
}

func TestUniqueActions_NotFailure(t *testing.T) {
	dir := t.TempDir()
	actions1 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "a"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc123", SizeBytes: 10}},
			},
		},
		{
			CommandArgs:   []string{"/bin/echo", "only-in-1"},
			ListedOutputs: []string{"out/only1.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/only1.txt", Digest: &pb.Digest{Hash: "111", SizeBytes: 5}},
			},
		},
	}
	actions2 := []*pb.SpawnExec{
		{
			CommandArgs:   []string{"/bin/echo", "a"},
			ListedOutputs: []string{"out/a.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc123", SizeBytes: 10}},
			},
		},
		{
			CommandArgs:   []string{"/bin/echo", "only-in-2"},
			ListedOutputs: []string{"out/only2.txt"},
			Remotable:     true,
			Cacheable:     true,
			Mnemonic:      "Genrule",
			ActualOutputs: []*pb.File{
				{Path: "out/only2.txt", Digest: &pb.Digest{Hash: "222", SizeBytes: 5}},
			},
		},
	}
	log1 := writeLogs(t, dir, "log1.bin", actions1)
	log2 := writeLogs(t, dir, "log2.bin", actions2)

	code := run([]string{log1, log2}, "")
	if code != exitDeterministic {
		t.Errorf("unique actions (no paired diffs): got exit code %d, want %d", code, exitDeterministic)
	}
}

func TestDiffSections(t *testing.T) {
	a := &pb.SpawnExec{
		CommandArgs:   []string{"/bin/echo", "hello"},
		ListedOutputs: []string{"out/a.txt"},
		ActualOutputs: []*pb.File{
			{Path: "out/a.txt", Digest: &pb.Digest{Hash: "abc", SizeBytes: 10}},
		},
		Inputs: []*pb.File{
			{Path: "in/x.txt", Digest: &pb.Digest{Hash: "inp1", SizeBytes: 5}},
		},
	}
	b := &pb.SpawnExec{
		CommandArgs:   []string{"/bin/echo", "world"},
		ListedOutputs: []string{"out/a.txt"},
		ActualOutputs: []*pb.File{
			{Path: "out/a.txt", Digest: &pb.Digest{Hash: "def", SizeBytes: 10}},
		},
		Inputs: []*pb.File{
			{Path: "in/x.txt", Digest: &pb.Digest{Hash: "inp1", SizeBytes: 5}},
		},
	}

	diffs := diffSections(a, b)

	// Expect command_args and actual_outputs to differ.
	wantSections := map[string]bool{
		"command_args":   true,
		"actual_outputs": true,
	}
	gotSections := make(map[string]bool)
	for _, s := range diffs {
		gotSections[s] = true
	}

	for want := range wantSections {
		if !gotSections[want] {
			t.Errorf("expected section %q in diffs, got %v", want, diffs)
		}
	}

	// Ensure sections that are the same are not reported.
	notExpected := []string{"inputs", "listed_outputs", "environment_variables", "platform"}
	for _, ne := range notExpected {
		if gotSections[ne] {
			t.Errorf("section %q should not be in diffs, got %v", ne, diffs)
		}
	}
}

func TestWrongArgCount_Exit2(t *testing.T) {
	code := run([]string{"/nonexistent"}, "")
	if code != exitUsageError {
		t.Errorf("wrong arg count: got exit code %d, want %d", code, exitUsageError)
	}
}
