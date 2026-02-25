package ui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value", "hello": "world"}
	if err := PrintJSON(data); err != nil {
		t.Fatalf("PrintJSON returned error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("PrintJSON output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if result["key"] != "value" || result["hello"] != "world" {
		t.Errorf("unexpected result: %v", result)
	}
}
