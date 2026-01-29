package execlog

import (
	"bytes"
	"testing"

	pb "tools/execlog/proto"
)

func TestQuoteString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", `"hello"`},
		{`path/to/file.txt`, `"path/to/file.txt"`},
		{"line1\nline2", `"line1\nline2"`},
		{`back\slash`, `"back\\slash"`},
		{`say "hi"`, `"say \"hi\""`},
		{"tab\there", `"tab\there"`},
		{"cr\rhere", `"cr\rhere"`},
		{"\x01\x02", `"\001\002"`},
		{"", `""`},
	}
	for _, tt := range tests {
		got := quoteString(tt.input)
		if got != tt.want {
			t.Errorf("quoteString(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestFormatSpawnExec_ZeroValueOmission(t *testing.T) {
	exec := &pb.SpawnExec{}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for zero-value SpawnExec, got:\n%s", buf.String())
	}
}

func TestFormatSpawnExec_BoolFields(t *testing.T) {
	exec := &pb.SpawnExec{
		Remotable: true,
		Cacheable: true,
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "remotable: true\ncacheable: true\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSpawnExec_FalseBoolOmitted(t *testing.T) {
	exec := &pb.SpawnExec{
		Remotable: false,
		Cacheable: false,
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("false bools should be omitted, got:\n%s", buf.String())
	}
}

func TestFormatSpawnExec_FullMessage(t *testing.T) {
	exec := &pb.SpawnExec{
		CommandArgs: []string{"/bin/bash", "-c", "echo hello"},
		EnvironmentVariables: []*pb.EnvironmentVariable{
			{Name: "PATH", Value: "/usr/bin"},
		},
		Inputs: []*pb.File{
			{
				Path: "foo/bar.txt",
				Digest: &pb.Digest{
					Hash:             "abc123",
					SizeBytes:        42,
					HashFunctionName: "SHA-256",
				},
			},
		},
		ListedOutputs: []string{"out/result.txt"},
		Remotable:     true,
		Cacheable:     true,
		Mnemonic:      "Genrule",
		ActualOutputs: []*pb.File{
			{
				Path: "out/result.txt",
				Digest: &pb.Digest{
					Hash:      "def456",
					SizeBytes: 10,
				},
			},
		},
		Runner: "linux-sandbox",
		Status: "success",
	}

	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}

	want := `command_args: "/bin/bash"
command_args: "-c"
command_args: "echo hello"
environment_variables {
  name: "PATH"
  value: "/usr/bin"
}
inputs {
  path: "foo/bar.txt"
  digest {
    hash: "abc123"
    size_bytes: 42
    hash_function_name: "SHA-256"
  }
}
listed_outputs: "out/result.txt"
remotable: true
cacheable: true
mnemonic: "Genrule"
actual_outputs {
  path: "out/result.txt"
  digest {
    hash: "def456"
    size_bytes: 10
  }
}
runner: "linux-sandbox"
status: "success"
`
	got := buf.String()
	if got != want {
		t.Errorf("FormatSpawnExec mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSpawnExec_Platform(t *testing.T) {
	exec := &pb.SpawnExec{
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{Name: "OSFamily", Value: "Linux"},
			},
		},
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	want := `platform {
  properties {
    name: "OSFamily"
    value: "Linux"
  }
}
`
	got := buf.String()
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSpawnExec_EmptyPlatformOmitted(t *testing.T) {
	exec := &pb.SpawnExec{
		Platform: &pb.Platform{},
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("empty platform should be omitted, got:\n%s", buf.String())
	}
}

func TestFormatSpawnExec_ExitCodeZeroOmitted(t *testing.T) {
	exec := &pb.SpawnExec{
		ExitCode: 0,
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("exit_code 0 should be omitted, got:\n%s", buf.String())
	}
}

func TestFormatSpawnExec_ExitCodeNonZero(t *testing.T) {
	exec := &pb.SpawnExec{
		ExitCode: 1,
	}
	var buf bytes.Buffer
	if err := FormatSpawnExec(&buf, exec); err != nil {
		t.Fatal(err)
	}
	want := "exit_code: 1\n"
	got := buf.String()
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
