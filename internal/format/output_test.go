package format_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebmish/ynab-cli/internal/format"
)

func TestJSONFormat(t *testing.T) {
	input := []byte(`{"id":1,"name":"test"}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var v interface{}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if !strings.Contains(buf.String(), "\n") {
		t.Error("expected pretty-printed")
	}
}

func TestEnvelopeUnwrap(t *testing.T) {
	input := []byte(`{"data":{"user":{"id":"u1"}}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, `"data"`) {
		t.Errorf("expected envelope unwrapped, got %s", out)
	}
	if !strings.Contains(out, "u1") {
		t.Errorf("expected inner data preserved, got %s", out)
	}
}

func TestEnvelopeUnwrapThenFields(t *testing.T) {
	input := []byte(`{"data":{"account":{"id":"a1","name":"Checking","balance":1000}}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json", Fields: []string{"id", "name"}}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, `"data"`) || strings.Contains(out, `"account"`) {
		t.Errorf("envelope not fully unwrapped: %s", out)
	}
	if !strings.Contains(out, `"id"`) || !strings.Contains(out, `"name"`) {
		t.Errorf("expected id and name fields: %s", out)
	}
	if strings.Contains(out, "balance") {
		t.Errorf("balance should be filtered out: %s", out)
	}
}

func TestNoUnwrapWhenMultipleKeys(t *testing.T) {
	input := []byte(`{"data":{"id":1},"meta":{"x":1}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "data") || !strings.Contains(buf.String(), "meta") {
		t.Errorf("should not unwrap when object has multiple keys: %s", buf.String())
	}
}

func TestNDJSONEnvelopeWithSingleArrayKey(t *testing.T) {
	input := []byte(`{"data":{"transactions":[{"id":"t1"},{"id":"t2"},{"id":"t3"}]}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 ndjson lines, got %d: %q", len(lines), buf.String())
	}
	for i, l := range lines {
		var v interface{}
		if err := json.Unmarshal([]byte(l), &v); err != nil {
			t.Errorf("line %d invalid JSON: %v", i, err)
		}
	}
}

func TestNDJSONArray(t *testing.T) {
	input := []byte(`[{"id":1},{"id":2}]`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestNDJSONSingleObject(t *testing.T) {
	input := []byte(`{"data":{"user":{"id":"u1"}}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %q", len(lines), buf.String())
	}
}

func TestFieldsNestedPath(t *testing.T) {
	// Envelope unwrap peels both `data` and the single inner key `category_groups`,
	// so by the time fields run, the value is an array of group objects. Fields are
	// relative to the post-unwrap shape — this matches the documented convention
	// (see ynab-shared SKILL.md: "target inner fields directly").
	input := []byte(`{"data":{"category_groups":[` +
		`{"id":"g1","name":"Bills","categories":[{"id":"c1","name":"Rent","balance":100},{"id":"c2","name":"Water","balance":50}]},` +
		`{"id":"g2","name":"Food","categories":[{"id":"c3","name":"Groceries","balance":200}]}` +
		`]}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json", Fields: []string{"categories.name"}}); err != nil {
		t.Fatal(err)
	}
	var out interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Expect: [{categories:[{name:Rent},{name:Water}]},{categories:[{name:Groceries}]}]
	arr, ok := out.([]interface{})
	if !ok {
		t.Fatalf("expected top-level array, got %T: %s", out, buf.String())
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(arr))
	}
	first := arr[0].(map[string]interface{})
	if _, hasName := first["name"]; hasName {
		t.Errorf("group name should be filtered out: %s", buf.String())
	}
	if _, hasID := first["id"]; hasID {
		t.Errorf("group id should be filtered out: %s", buf.String())
	}
	cats, ok := first["categories"].([]interface{})
	if !ok {
		t.Fatalf("expected categories array in first group")
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories in first group, got %d", len(cats))
	}
	cat0 := cats[0].(map[string]interface{})
	if cat0["name"] != "Rent" {
		t.Errorf("expected name=Rent, got %v", cat0["name"])
	}
	if _, hasBal := cat0["balance"]; hasBal {
		t.Errorf("balance should be filtered out: %s", buf.String())
	}
	if _, hasID := cat0["id"]; hasID {
		t.Errorf("category id should be filtered out: %s", buf.String())
	}
}

func TestFieldsMixedTopLevelAndNested(t *testing.T) {
	input := []byte(`[{"id":"a","meta":{"x":1,"y":2},"drop":"z"},{"id":"b","meta":{"x":3,"y":4},"drop":"z"}]`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json", Fields: []string{"id", "meta.x"}}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "drop") {
		t.Errorf("drop field should be filtered: %s", out)
	}
	if strings.Contains(out, `"y"`) {
		t.Errorf("meta.y should be filtered: %s", out)
	}
	if !strings.Contains(out, `"id"`) || !strings.Contains(out, `"meta"`) || !strings.Contains(out, `"x"`) {
		t.Errorf("expected id + meta.x preserved: %s", out)
	}
}

func TestFieldsLeafKeepsWholeSubtree(t *testing.T) {
	input := []byte(`{"data":{"account":{"id":"a1","sub":{"x":1,"y":2},"other":"z"}}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json", Fields: []string{"sub"}}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"x"`) || !strings.Contains(out, `"y"`) {
		t.Errorf("leaf path should keep whole subtree: %s", out)
	}
	if strings.Contains(out, "other") || strings.Contains(out, `"id"`) {
		t.Errorf("non-selected keys should be filtered: %s", out)
	}
}

func TestFieldsUnknownPathYieldsEmpty(t *testing.T) {
	// Filtering by a path that doesn't exist should yield empty objects, not an error.
	// This preserves today's "silent empty" behavior when the path is genuinely absent
	// — only bogus *nested* paths used to be the silent-failure mode, and that's fixed.
	input := []byte(`{"id":"a","name":"n"}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "json", Fields: []string{"does_not_exist"}}); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "{}" {
		t.Errorf("expected empty object for unknown field, got %q", out)
	}
}

func TestFlattenSplits(t *testing.T) {
	// Mixed: one split with 2 subs, one non-split. Splits carry full parent context
	// on each emitted record; non-split passes through.
	input := []byte(`{"data":{"transactions":[
		{"id":"p1","date":"2025-02-01","account_id":"a1","account_name":"HSBC","payee_name":"Payroll","cleared":"cleared","approved":true,"amount":5000000,"category_name":"Split","category_id":"split-id","subtransactions":[
			{"id":"s1","transaction_id":"p1","amount":-1500000,"category_id":"tax","category_name":"Tax","memo":"PAYE","deleted":false},
			{"id":"s2","transaction_id":"p1","amount":6500000,"category_id":"rta","category_name":"Inflow: Ready to Assign","memo":null,"deleted":false}
		]},
		{"id":"t2","date":"2025-02-03","account_id":"a1","account_name":"HSBC","payee_name":"Tesco","amount":-25000,"category_id":"groc","category_name":"Groceries"}
	]}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson", FlattenSplits: true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 records (2 subs + 1 non-split), got %d: %s", len(lines), buf.String())
	}
	var sub1, sub2, other map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &sub1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &sub2); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[2]), &other); err != nil {
		t.Fatal(err)
	}

	// Sub 1 should carry parent date/account but own category and amount.
	if sub1["date"] != "2025-02-01" {
		t.Errorf("sub1: expected inherited date, got %v", sub1["date"])
	}
	if sub1["account_name"] != "HSBC" {
		t.Errorf("sub1: expected inherited account_name, got %v", sub1["account_name"])
	}
	if sub1["category_name"] != "Tax" {
		t.Errorf("sub1: expected overridden category_name=Tax, got %v", sub1["category_name"])
	}
	if sub1["amount"].(float64) != -1500000 {
		t.Errorf("sub1: expected own amount, got %v", sub1["amount"])
	}
	if sub1["id"] != "s1" {
		t.Errorf("sub1: expected subtxn id, got %v", sub1["id"])
	}
	if sub1["transaction_id"] != "p1" {
		t.Errorf("sub1: expected transaction_id=p1 (parent ref), got %v", sub1["transaction_id"])
	}
	if _, has := sub1["subtransactions"]; has {
		t.Errorf("sub1: subtransactions should not appear on flattened record")
	}
	if sub1["approved"] != true {
		t.Errorf("sub1: expected inherited approved=true, got %v", sub1["approved"])
	}

	// Sub 2 has memo:null — should fall back to parent (parent has no memo, so absent/null).
	if sub2["category_name"] != "Inflow: Ready to Assign" {
		t.Errorf("sub2: category_name wrong, got %v", sub2["category_name"])
	}
	if sub2["amount"].(float64) != 6500000 {
		t.Errorf("sub2: amount wrong, got %v", sub2["amount"])
	}

	// Non-split passes through.
	if other["id"] != "t2" {
		t.Errorf("other: id should be t2, got %v", other["id"])
	}
	if other["category_name"] != "Groceries" {
		t.Errorf("other: category_name should be Groceries, got %v", other["category_name"])
	}
}

func TestFlattenSplitsSkipsDeletedSubs(t *testing.T) {
	input := []byte(`{"data":{"transactions":[
		{"id":"p1","date":"2025-01-01","amount":0,"subtransactions":[
			{"id":"s1","transaction_id":"p1","amount":100,"deleted":false},
			{"id":"s2","transaction_id":"p1","amount":200,"deleted":true}
		]}
	]}}`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson", FlattenSplits: true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 record (deleted sub dropped), got %d: %s", len(lines), buf.String())
	}
}

func TestFlattenSplitsNoOpWhenNoSplits(t *testing.T) {
	input := []byte(`[{"id":"t1","amount":100},{"id":"t2","amount":200}]`)
	var buf bytes.Buffer
	if err := format.Write(&buf, input, format.Options{Format: "ndjson", FlattenSplits: true}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines unchanged, got %d", len(lines))
	}
}

func TestRawBinary(t *testing.T) {
	data := []byte{0x01, 0x02, 0xFF}
	var buf bytes.Buffer
	if err := format.WriteRaw(&buf, data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("raw bytes mismatch")
	}
}
