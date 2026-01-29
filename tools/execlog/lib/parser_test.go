package execlog

import (
	"bufio"
	"bytes"
	"testing"

	pb "tools/execlog/proto"
	"google.golang.org/protobuf/encoding/protodelim"
)

// writeDelimited writes a varint-delimited SpawnExec to buf.
func writeDelimited(t *testing.T, buf *bytes.Buffer, exec *pb.SpawnExec) {
	t.Helper()
	if _, err := protodelim.MarshalTo(buf, exec); err != nil {
		t.Fatal(err)
	}
}

func TestFilteringParser_NoFilter(t *testing.T) {
	var buf bytes.Buffer
	e1 := &pb.SpawnExec{Mnemonic: "Genrule", Runner: "linux-sandbox"}
	e2 := &pb.SpawnExec{Mnemonic: "CppCompile", Runner: "remote"}
	writeDelimited(t, &buf, e1)
	writeDelimited(t, &buf, e2)

	parser := NewFilteringParser(&buf, "")

	got1, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got1.Mnemonic != "Genrule" {
		t.Errorf("got mnemonic %q, want %q", got1.Mnemonic, "Genrule")
	}

	got2, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got2.Mnemonic != "CppCompile" {
		t.Errorf("got mnemonic %q, want %q", got2.Mnemonic, "CppCompile")
	}

	got3, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got3 != nil {
		t.Errorf("expected nil at EOF, got %v", got3)
	}
}

func TestFilteringParser_WithFilter(t *testing.T) {
	var buf bytes.Buffer
	e1 := &pb.SpawnExec{Mnemonic: "Genrule", Runner: "linux-sandbox"}
	e2 := &pb.SpawnExec{Mnemonic: "CppCompile", Runner: "remote"}
	e3 := &pb.SpawnExec{Mnemonic: "Action", Runner: "linux-sandbox"}
	writeDelimited(t, &buf, e1)
	writeDelimited(t, &buf, e2)
	writeDelimited(t, &buf, e3)

	parser := NewFilteringParser(&buf, "linux-sandbox")

	got1, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got1.Mnemonic != "Genrule" {
		t.Errorf("got mnemonic %q, want %q", got1.Mnemonic, "Genrule")
	}

	got2, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got2.Mnemonic != "Action" {
		t.Errorf("got mnemonic %q, want %q", got2.Mnemonic, "Action")
	}

	got3, err := parser.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got3 != nil {
		t.Errorf("expected nil at EOF, got %v", got3)
	}
}

func TestReadSpawnExec_EOF(t *testing.T) {
	var buf bytes.Buffer
	reader := bufio.NewReader(&buf)
	exec, err := ReadSpawnExec(reader)
	if err != nil {
		t.Fatal(err)
	}
	if exec != nil {
		t.Errorf("expected nil at EOF, got %v", exec)
	}
}

func TestGolden(t *testing.T) {
	g := NewGolden()

	e1 := &pb.SpawnExec{ListedOutputs: []string{"out/a.txt"}}
	e2 := &pb.SpawnExec{ListedOutputs: []string{"out/b.txt"}}
	e3 := &pb.SpawnExec{} // no outputs

	g.AddSpawnExec(e1)
	g.AddSpawnExec(e2)
	g.AddSpawnExec(e3)

	if pos := g.PositionFor(e1); pos != 0 {
		t.Errorf("e1 position = %d, want 0", pos)
	}
	if pos := g.PositionFor(e2); pos != 1 {
		t.Errorf("e2 position = %d, want 1", pos)
	}
	if pos := g.PositionFor(e3); pos != -1 {
		t.Errorf("e3 position = %d, want -1", pos)
	}

	unknown := &pb.SpawnExec{ListedOutputs: []string{"out/unknown.txt"}}
	if pos := g.PositionFor(unknown); pos != -1 {
		t.Errorf("unknown position = %d, want -1", pos)
	}
}

func TestReorderingParser(t *testing.T) {
	// Golden order: a, b, c
	golden := NewGolden()
	golden.AddSpawnExec(&pb.SpawnExec{ListedOutputs: []string{"out/a.txt"}})
	golden.AddSpawnExec(&pb.SpawnExec{ListedOutputs: []string{"out/b.txt"}})
	golden.AddSpawnExec(&pb.SpawnExec{ListedOutputs: []string{"out/c.txt"}})

	// Second file order: c, b, a, plus one unique
	var buf bytes.Buffer
	writeDelimited(t, &buf, &pb.SpawnExec{ListedOutputs: []string{"out/c.txt"}, Mnemonic: "C"})
	writeDelimited(t, &buf, &pb.SpawnExec{ListedOutputs: []string{"out/b.txt"}, Mnemonic: "B"})
	writeDelimited(t, &buf, &pb.SpawnExec{ListedOutputs: []string{"out/a.txt"}, Mnemonic: "A"})
	writeDelimited(t, &buf, &pb.SpawnExec{ListedOutputs: []string{"out/d.txt"}, Mnemonic: "D"})

	input := NewFilteringParser(&buf, "")
	rp, err := NewReorderingParser(golden, input)
	if err != nil {
		t.Fatal(err)
	}

	// Should come back in golden order: A, B, C, then unique: D
	expected := []string{"A", "B", "C", "D"}
	for i, want := range expected {
		exec, err := rp.Next()
		if err != nil {
			t.Fatal(err)
		}
		if exec == nil {
			t.Fatalf("unexpected nil at index %d", i)
		}
		if exec.Mnemonic != want {
			t.Errorf("index %d: got mnemonic %q, want %q", i, exec.Mnemonic, want)
		}
	}

	// Should be done
	exec, err := rp.Next()
	if err != nil {
		t.Fatal(err)
	}
	if exec != nil {
		t.Errorf("expected nil at end, got %v", exec)
	}
}
