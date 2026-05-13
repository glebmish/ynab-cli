package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSkillsListTextE2E(t *testing.T) {
	rootCmd.SetArgs([]string{"skills", "list"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skills list failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"ynab-shared", "ynab-budgeting", "ynab-transactions"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("skills list output missing %q\noutput:\n%s", want, got)
		}
	}
}

func TestSkillsListJSONE2E(t *testing.T) {
	rootCmd.SetArgs([]string{"skills", "list", "--format", "json"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skills list --format json failed: %v", err)
	}
	var entries []map[string]any
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("output not valid JSON: %v\noutput:\n%s", err, out.String())
	}
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 skill entries, got %d", len(entries))
	}
	for _, e := range entries {
		if _, ok := e["name"].(string); !ok {
			t.Errorf("entry missing string 'name': %v", e)
		}
		if _, ok := e["description"].(string); !ok {
			t.Errorf("entry missing string 'description': %v", e)
		}
	}
}

func TestSkillsGetRawE2E(t *testing.T) {
	// Reset persistent --format to default before running, since prior tests
	// may have marked it changed.
	rootCmd.PersistentFlags().Set("format", "json")
	rootCmd.PersistentFlags().Lookup("format").Changed = false

	rootCmd.SetArgs([]string{"skills", "get", "ynab-shared"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skills get failed: %v", err)
	}
	first := bytes.SplitN(out.Bytes(), []byte("\n"), 2)[0]
	if bytes.Equal(first, []byte("---")) {
		t.Errorf("skills get raw output starts with frontmatter delimiter; want body only.\noutput head:\n%s", string(first))
	}
	if out.Len() == 0 {
		t.Errorf("skills get raw output is empty")
	}
}

func TestSkillsGetUnknownE2E(t *testing.T) {
	rootCmd.SetArgs([]string{"skills", "get", "this-skill-does-not-exist"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown skill")
	}
}
