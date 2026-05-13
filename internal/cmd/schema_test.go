package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestAllOperationIdsMapped: every operationId in the spec must appear in
// operationIDToCommand. Catches spec drift — a new endpoint added to the
// spec without a CLI mapping.
func TestAllOperationIdsMapped(t *testing.T) {
	spec, err := parseSpec()
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := spec["paths"].(map[string]interface{})
	var missing []string
	for _, item := range paths {
		pi, _ := item.(map[string]interface{})
		for _, m := range []string{"get", "post", "put", "delete", "patch"} {
			op, ok := pi[m].(map[string]interface{})
			if !ok {
				continue
			}
			opID, _ := op["operationId"].(string)
			if opID == "" {
				continue
			}
			if _, ok := operationIDToCommand[opID]; !ok {
				missing = append(missing, opID)
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d operationIds missing CLI mapping: %v", len(missing), missing)
	}
}

// TestEveryMappedCLINameExists: every value in operationIDToCommand must
// resolve to a real cobra leaf command. Catches stale map entries pointing
// at commands that have been renamed or removed.
func TestEveryMappedCLINameExists(t *testing.T) {
	var missing []string
	for opID, cliName := range operationIDToCommand {
		parts := strings.SplitN(cliName, ".", 2)
		if len(parts) != 2 {
			missing = append(missing, cliName+" (malformed name)")
			continue
		}
		var group *cobra.Command
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == parts[0] {
				group = sub
				break
			}
		}
		if group == nil {
			missing = append(missing, cliName+" (group not found, from "+opID+")")
			continue
		}
		found := false
		for _, sub := range group.Commands() {
			if sub.Name() == parts[1] {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, cliName+" (from "+opID+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d mapped CLI names not registered: %v", len(missing), missing)
	}
}

// TestResolveRefs is a small unit test for the $ref inliner.
func TestResolveRefs(t *testing.T) {
	spec, err := parseSpec()
	if err != nil {
		t.Fatal(err)
	}
	components, _ := spec["components"].(map[string]interface{})
	schemas, _ := components["schemas"].(map[string]interface{})
	obj := map[string]interface{}{
		"$ref": "#/components/schemas/User",
	}
	resolveAllRefs(obj, schemas, map[string]bool{})
	if _, still := obj["$ref"]; still && len(obj) == 1 {
		t.Errorf("expected $ref to be inlined, got %v", obj)
	}
}
