package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

	code := run([]string{log1, log2}, "", false)
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

	code := run([]string{log1, log2}, "", false)
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

	code := run([]string{log1, log2}, "", false)
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

	code := run([]string{log1, log2}, "", false)
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

	notExpected := []string{"inputs", "listed_outputs", "environment_variables", "platform"}
	for _, ne := range notExpected {
		if gotSections[ne] {
			t.Errorf("section %q should not be in diffs, got %v", ne, diffs)
		}
	}
}

func TestWrongArgCount_Exit2(t *testing.T) {
	code := run([]string{"/nonexistent"}, "", false)
	if code != exitUsageError {
		t.Errorf("wrong arg count: got exit code %d, want %d", code, exitUsageError)
	}
}

func TestVerboseDetails(t *testing.T) {
	a := &pb.SpawnExec{
		CommandArgs: []string{"/bin/echo", "hello"},
		EnvironmentVariables: []*pb.EnvironmentVariable{
			{Name: "PATH", Value: "/usr/bin"},
			{Name: "HOME", Value: "/home/user"},
		},
		Inputs: []*pb.File{
			{Path: "in/x.txt", Digest: &pb.Digest{Hash: "aaa", SizeBytes: 10}},
			{Path: "in/y.txt", Digest: &pb.Digest{Hash: "bbb", SizeBytes: 20}},
		},
		ActualOutputs: []*pb.File{
			{Path: "out/a.txt", Digest: &pb.Digest{Hash: "ooo", SizeBytes: 5}},
		},
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{Name: "OSFamily", Value: "Linux"},
			},
		},
	}
	b := &pb.SpawnExec{
		CommandArgs: []string{"/bin/echo", "world", "--flag"},
		EnvironmentVariables: []*pb.EnvironmentVariable{
			{Name: "PATH", Value: "/usr/local/bin"},
			{Name: "LANG", Value: "en_US"},
		},
		Inputs: []*pb.File{
			{Path: "in/x.txt", Digest: &pb.Digest{Hash: "aaa2", SizeBytes: 10}},
			{Path: "in/z.txt", Digest: &pb.Digest{Hash: "ccc", SizeBytes: 30}},
		},
		ActualOutputs: []*pb.File{
			{Path: "out/a.txt", Digest: &pb.Digest{Hash: "ppp", SizeBytes: 5}},
		},
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{Name: "OSFamily", Value: "Darwin"},
			},
		},
	}

	t.Run("command_args", func(t *testing.T) {
		lines := verboseDetails("command_args", a, b)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, `"hello" -> "world"`) {
			t.Errorf("expected changed arg, got:\n%s", joined)
		}
		if !strings.Contains(joined, `"--flag"`) {
			t.Errorf("expected added arg, got:\n%s", joined)
		}
	})

	t.Run("environment_variables", func(t *testing.T) {
		lines := verboseDetails("environment_variables", a, b)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "PATH") {
			t.Errorf("expected PATH change, got:\n%s", joined)
		}
		if !strings.Contains(joined, "HOME") {
			t.Errorf("expected HOME removed, got:\n%s", joined)
		}
		if !strings.Contains(joined, "LANG") {
			t.Errorf("expected LANG added, got:\n%s", joined)
		}
	})

	t.Run("inputs", func(t *testing.T) {
		lines := verboseDetails("inputs", a, b)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "in/x.txt") {
			t.Errorf("expected in/x.txt changed, got:\n%s", joined)
		}
		if !strings.Contains(joined, "in/y.txt") {
			t.Errorf("expected in/y.txt removed, got:\n%s", joined)
		}
		if !strings.Contains(joined, "in/z.txt") {
			t.Errorf("expected in/z.txt added, got:\n%s", joined)
		}
	})

	t.Run("actual_outputs", func(t *testing.T) {
		lines := verboseDetails("actual_outputs", a, b)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "out/a.txt") {
			t.Errorf("expected out/a.txt changed, got:\n%s", joined)
		}
		if !strings.Contains(joined, "ooo") || !strings.Contains(joined, "ppp") {
			t.Errorf("expected hash values in output, got:\n%s", joined)
		}
	})

	t.Run("platform", func(t *testing.T) {
		lines := verboseDetails("platform", a, b)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "OSFamily") {
			t.Errorf("expected OSFamily change, got:\n%s", joined)
		}
		if !strings.Contains(joined, "Linux") || !strings.Contains(joined, "Darwin") {
			t.Errorf("expected old/new values, got:\n%s", joined)
		}
	})
}
