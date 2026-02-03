package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrinter_Table_DefaultFormat(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatTable}

	headers := []string{"NAME", "AGE"}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}
	p.Table(headers, rows)

	result := out.String()
	if !strings.Contains(result, "NAME") {
		t.Errorf("expected header NAME in output, got: %s", result)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("expected Alice in output, got: %s", result)
	}
	if !strings.Contains(result, "Bob") {
		t.Errorf("expected Bob in output, got: %s", result)
	}
	// Verify separator exists
	if !strings.Contains(result, "---") {
		t.Errorf("expected separator in output, got: %s", result)
	}
}

func TestPrinter_Table_JSONFormat(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatJSON}

	headers := []string{"name", "age"}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}
	p.Table(headers, rows)

	var records []map[string]string
	if err := json.Unmarshal(out.Bytes(), &records); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, out.String())
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %q", records[0]["name"])
	}
	if records[1]["age"] != "25" {
		t.Errorf("expected 25, got %q", records[1]["age"])
	}
}

func TestPrinter_Table_QuietFormat(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatQuiet}

	headers := []string{"ID", "NAME"}
	rows := [][]string{
		{"123", "Alice"},
		{"456", "Bob"},
	}
	p.Table(headers, rows)

	result := out.String()
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "123" {
		t.Errorf("expected '123', got %q", lines[0])
	}
	if lines[1] != "456" {
		t.Errorf("expected '456', got %q", lines[1])
	}
}

func TestPrinter_Success_Quiet(t *testing.T) {
	var errBuf bytes.Buffer
	p := &Printer{Out: &bytes.Buffer{}, Err: &errBuf, Format: FormatQuiet}
	p.Success("done")
	if errBuf.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got: %s", errBuf.String())
	}
}

func TestPrinter_Error_AlwaysShown(t *testing.T) {
	var errBuf bytes.Buffer
	p := &Printer{Out: &bytes.Buffer{}, Err: &errBuf, Format: FormatQuiet}
	p.Error("something broke")
	if !strings.Contains(errBuf.String(), "something broke") {
		t.Errorf("expected error message even in quiet mode, got: %s", errBuf.String())
	}
}

func TestPrinter_Info_Quiet(t *testing.T) {
	var errBuf bytes.Buffer
	p := &Printer{Out: &bytes.Buffer{}, Err: &errBuf, Format: FormatQuiet}
	p.Info("info msg")
	if errBuf.Len() != 0 {
		t.Errorf("expected no info in quiet mode, got: %s", errBuf.String())
	}
}

func TestPrinter_JSON(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatJSON}
	p.JSON(map[string]string{"key": "val"})

	var parsed map[string]string
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if parsed["key"] != "val" {
		t.Errorf("expected val, got %q", parsed["key"])
	}
}

func TestPrinter_Table_EmptyRows(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatTable}
	p.Table([]string{"A", "B"}, nil)

	result := out.String()
	if !strings.Contains(result, "A") {
		t.Errorf("expected header even with no rows, got: %s", result)
	}
}

func TestPrinter_Table_JSONEmpty(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Out: &out, Err: &bytes.Buffer{}, Format: FormatJSON}
	p.Table([]string{"A"}, nil)

	var records []map[string]string
	if err := json.Unmarshal(out.Bytes(), &records); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty array, got %d records", len(records))
	}
}
