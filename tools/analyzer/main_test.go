package main

import (
	"testing"
)

func TestParseLine_Separator(t *testing.T) {
	pl := parseLine("  ---------------------------------------------------------")
	if pl.typ != lineActionSeparator {
		t.Errorf("expected lineActionSeparator, got %d", pl.typ)
	}
}

func TestParseLine_Diff(t *testing.T) {
	tests := []struct {
		input string
		want  lineType
	}{
		{"1|  some content", lineDiff},
		{"2|  other content", lineDiff},
	}
	for _, tt := range tests {
		pl := parseLine(tt.input)
		if pl.typ != tt.want {
			t.Errorf("parseLine(%q).typ = %d, want %d", tt.input, pl.typ, tt.want)
		}
	}
}

func TestParseLine_Section(t *testing.T) {
	pl := parseLine("  inputs {")
	if pl.typ != lineSection {
		t.Errorf("expected lineSection, got %d", pl.typ)
	}
	if pl.stringPayload != "inputs" {
		t.Errorf("got payload %q, want %q", pl.stringPayload, "inputs")
	}
}

func TestParseLine_Mnemonic(t *testing.T) {
	pl := parseLine(`  mnemonic: "Genrule"`)
	if pl.typ != lineMnemonic {
		t.Errorf("expected lineMnemonic, got %d", pl.typ)
	}
	if pl.stringPayload != `"Genrule"` {
		t.Errorf("got payload %q, want %q", pl.stringPayload, `"Genrule"`)
	}
}

func TestParseLine_Remotable(t *testing.T) {
	pl := parseLine("  remotable: true")
	if pl.typ != lineRemotable {
		t.Errorf("expected lineRemotable, got %d", pl.typ)
	}
	if !pl.boolPayload {
		t.Error("expected boolPayload true")
	}
}

func TestParseLine_Cacheable(t *testing.T) {
	pl := parseLine("  cacheable: false")
	if pl.typ != lineCacheable {
		t.Errorf("expected lineCacheable, got %d", pl.typ)
	}
	if pl.boolPayload {
		t.Error("expected boolPayload false")
	}
}

func TestParseLine_Boring(t *testing.T) {
	tests := []string{
		"  }",
		`    name: "PATH"`,
		"",
	}
	for _, input := range tests {
		pl := parseLine(input)
		if pl.typ != lineBoring {
			t.Errorf("parseLine(%q).typ = %d, want lineBoring", input, pl.typ)
		}
	}
}

func TestParseLine_CacheableTrue(t *testing.T) {
	pl := parseLine("  cacheable: true")
	if pl.typ != lineCacheable {
		t.Errorf("expected lineCacheable, got %d", pl.typ)
	}
	if !pl.boolPayload {
		t.Error("expected boolPayload true")
	}
}

func TestParseLine_ActualOutputsSection(t *testing.T) {
	pl := parseLine("  actual_outputs {")
	if pl.typ != lineSection {
		t.Errorf("expected lineSection, got %d", pl.typ)
	}
	if pl.stringPayload != "actual_outputs" {
		t.Errorf("got payload %q, want %q", pl.stringPayload, "actual_outputs")
	}
}
