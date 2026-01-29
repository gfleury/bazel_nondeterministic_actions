package execlog

import (
	"fmt"
	"io"
	"strings"

	pb "tools/execlog/proto"
)

// FormatSpawnExec writes a SpawnExec message to w in deterministic text format
// matching Java's TextFormat output. Fields are written in proto field number
// order with proto3 zero-value omission.
func FormatSpawnExec(w io.Writer, exec *pb.SpawnExec) error {
	for _, arg := range exec.CommandArgs {
		if _, err := fmt.Fprintf(w, "command_args: %s\n", quoteString(arg)); err != nil {
			return err
		}
	}
	for _, env := range exec.EnvironmentVariables {
		if _, err := fmt.Fprint(w, "environment_variables {\n"); err != nil {
			return err
		}
		if err := writeStringField(w, "  ", "name", env.Name); err != nil {
			return err
		}
		if err := writeStringField(w, "  ", "value", env.Value); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, "}\n"); err != nil {
			return err
		}
	}
	if exec.Platform != nil && len(exec.Platform.Properties) > 0 {
		if _, err := fmt.Fprint(w, "platform {\n"); err != nil {
			return err
		}
		for _, prop := range exec.Platform.Properties {
			if _, err := fmt.Fprint(w, "  properties {\n"); err != nil {
				return err
			}
			if err := writeStringField(w, "    ", "name", prop.Name); err != nil {
				return err
			}
			if err := writeStringField(w, "    ", "value", prop.Value); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, "  }\n"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, "}\n"); err != nil {
			return err
		}
	}
	for _, input := range exec.Inputs {
		if err := formatFile(w, "inputs", "", input); err != nil {
			return err
		}
	}
	for _, output := range exec.ListedOutputs {
		if _, err := fmt.Fprintf(w, "listed_outputs: %s\n", quoteString(output)); err != nil {
			return err
		}
	}
	if err := writeBoolField(w, "", "remotable", exec.Remotable); err != nil {
		return err
	}
	if err := writeBoolField(w, "", "cacheable", exec.Cacheable); err != nil {
		return err
	}
	if err := writeInt64Field(w, "", "timeout_millis", exec.TimeoutMillis); err != nil {
		return err
	}
	if err := writeStringField(w, "", "progress_message", exec.ProgressMessage); err != nil {
		return err
	}
	if err := writeStringField(w, "", "mnemonic", exec.Mnemonic); err != nil {
		return err
	}
	for _, output := range exec.ActualOutputs {
		if err := formatFile(w, "actual_outputs", "", output); err != nil {
			return err
		}
	}
	if err := writeStringField(w, "", "runner", exec.Runner); err != nil {
		return err
	}
	if err := writeBoolField(w, "", "remote_cache_hit", exec.RemoteCacheHit); err != nil {
		return err
	}
	if err := writeStringField(w, "", "status", exec.Status); err != nil {
		return err
	}
	if err := writeInt32Field(w, "", "exit_code", exec.ExitCode); err != nil {
		return err
	}
	return nil
}

func formatFile(w io.Writer, fieldName, indent string, file *pb.File) error {
	if _, err := fmt.Fprintf(w, "%s%s {\n", indent, fieldName); err != nil {
		return err
	}
	innerIndent := indent + "  "
	if err := writeStringField(w, innerIndent, "path", file.Path); err != nil {
		return err
	}
	if file.Digest != nil {
		if err := formatDigest(w, innerIndent, file.Digest); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%s}\n", indent); err != nil {
		return err
	}
	return nil
}

func formatDigest(w io.Writer, indent string, digest *pb.Digest) error {
	if _, err := fmt.Fprintf(w, "%sdigest {\n", indent); err != nil {
		return err
	}
	innerIndent := indent + "  "
	if err := writeStringField(w, innerIndent, "hash", digest.Hash); err != nil {
		return err
	}
	if err := writeInt64Field(w, innerIndent, "size_bytes", digest.SizeBytes); err != nil {
		return err
	}
	if err := writeStringField(w, innerIndent, "hash_function_name", digest.HashFunctionName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s}\n", indent); err != nil {
		return err
	}
	return nil
}

func writeStringField(w io.Writer, indent, name, value string) error {
	if value == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s: %s\n", indent, name, quoteString(value))
	return err
}

func writeBoolField(w io.Writer, indent, name string, value bool) error {
	if !value {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s: true\n", indent, name)
	return err
}

func writeInt64Field(w io.Writer, indent, name string, value int64) error {
	if value == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s: %d\n", indent, name, value)
	return err
}

func writeInt32Field(w io.Writer, indent, name string, value int32) error {
	if value == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s: %d\n", indent, name, value)
	return err
}

// quoteString returns a double-quoted string with C-style escaping matching
// Java's protobuf TextFormat output.
func quoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			if c < 0x20 || c == 0x7f {
				// Octal escape for non-printable characters (matches Java TextFormat)
				fmt.Fprintf(&b, "\\%03o", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
