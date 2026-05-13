package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Options struct {
	Format        string
	Fields        []string
	FlattenSplits bool
}

func FormatFromFlag(formatFlag, fieldsFlag string) Options {
	return FormatFromFlags(formatFlag, fieldsFlag, false)
}

func FormatFromFlags(formatFlag, fieldsFlag string, flattenSplits bool) Options {
	opts := Options{Format: formatFlag, FlattenSplits: flattenSplits}
	if fieldsFlag != "" {
		for _, f := range strings.Split(fieldsFlag, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				opts.Fields = append(opts.Fields, f)
			}
		}
	}
	return opts
}

// UnwrapBody parses a raw YNAB response body and returns the unwrapped value.
// Applies the same envelope peeling as Write (strips {"data":...},
// drops server_knowledge, peels single-key inner objects).
func UnwrapBody(data []byte) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return unwrapEnvelope(v), nil
}

// unwrapEnvelope strips the YNAB response envelope.
// 1. Peels outer {"data": X}.
// 2. Drops "server_knowledge" (YNAB sync metadata).
// 3. If one key remains (e.g. "plan", "accounts", "transaction"), peels it too.
func unwrapEnvelope(v interface{}) interface{} {
	if m, ok := v.(map[string]interface{}); ok && len(m) == 1 {
		if inner, ok := m["data"]; ok {
			v = inner
		}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return v
	}
	if _, has := m["server_knowledge"]; has {
		trimmed := make(map[string]interface{}, len(m)-1)
		for k, val := range m {
			if k != "server_knowledge" {
				trimmed[k] = val
			}
		}
		m = trimmed
		v = trimmed
	}
	if len(m) == 1 {
		for _, inner := range m {
			return inner
		}
	}
	return v
}

func Write(w io.Writer, data []byte, opts Options) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	v = unwrapEnvelope(v)
	v = sanitize(v)

	if opts.FlattenSplits {
		v = flattenSplits(v)
	}

	if len(opts.Fields) > 0 {
		v = filterFields(v, opts.Fields)
	}

	switch opts.Format {
	case "ndjson":
		return writeNDJSON(w, v)
	case "json", "text", "":
		return writePrettyJSON(w, v)
	default:
		return writePrettyJSON(w, v)
	}
}

// injectionTagPattern matches XML-ish tag wrappers used in some prompt-injection
// attempts, e.g. <system>...</system>, <assistant>...</assistant>. We strip the
// wrapper and keep the inner text — opening/closing tags only, case-insensitive.
var injectionTagPattern = regexp.MustCompile(`(?i)</?(system|assistant|tool_use|tool_result)[^>]*>`)

// sanitize walks parsed JSON and cleans string values to defend against
// prompt injection embedded in API responses. Strips control characters and
// known injection tag wrappers. Defensive minimum per design-cli §13.
func sanitize(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return sanitizeString(val)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, child := range val {
			out[k] = sanitize(child)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, child := range val {
			out[i] = sanitize(child)
		}
		return out
	default:
		return v
	}
}

func sanitizeString(s string) string {
	if s == "" {
		return s
	}
	s = injectionTagPattern.ReplaceAllString(s, "")
	if !needsControlStrip(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func needsControlStrip(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

func WriteRaw(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	return err
}

func DryRunOutput(w io.Writer, dryRunText string) error {
	_, err := fmt.Fprintln(w, dryRunText)
	return err
}

func writePrettyJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func writeNDJSON(w io.Writer, v interface{}) error {
	// If v is a map with exactly one array-valued key, stream that array.
	if m, ok := v.(map[string]interface{}); ok {
		var arrKey string
		arrCount := 0
		for k, val := range m {
			if _, isArr := val.([]interface{}); isArr {
				arrKey = k
				arrCount++
			}
		}
		if arrCount == 1 && len(m) == 1 {
			arr := m[arrKey].([]interface{})
			for _, elem := range arr {
				if err := writeJSONLine(w, elem); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if arr, ok := v.([]interface{}); ok {
		for _, elem := range arr {
			if err := writeJSONLine(w, elem); err != nil {
				return err
			}
		}
		return nil
	}

	return writeJSONLine(w, v)
}

func writeJSONLine(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Write(b)
	buf.WriteByte('\n')
	_, err = w.Write(buf.Bytes())
	return err
}

// flattenSplits expands split transactions so each subtransaction becomes its own
// top-level record, inheriting parent fields (date, account, cleared, approved, etc).
// Subtransaction fields override parent fields where both are set, except that a null
// subtransaction field falls back to the parent's value (YNAB semantics: null on a
// subtxn means "same as parent"). Non-split transactions pass through unchanged.
// Deleted subtransactions are dropped.
//
// Operates on array responses (e.g. transactions list). For non-array inputs (single
// get, other resources) it's a no-op.
func flattenSplits(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		return v
	}
	out := make([]interface{}, 0, len(arr))
	for _, elem := range arr {
		out = append(out, expandSplit(elem)...)
	}
	return out
}

func expandSplit(elem interface{}) []interface{} {
	m, ok := elem.(map[string]interface{})
	if !ok {
		return []interface{}{elem}
	}
	subs, _ := m["subtransactions"].([]interface{})
	if len(subs) == 0 {
		return []interface{}{elem}
	}
	parent := make(map[string]interface{}, len(m))
	for k, val := range m {
		if k == "subtransactions" {
			continue
		}
		parent[k] = val
	}
	out := make([]interface{}, 0, len(subs))
	for _, s := range subs {
		sm, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		if d, _ := sm["deleted"].(bool); d {
			continue
		}
		record := make(map[string]interface{}, len(parent)+len(sm))
		for k, val := range parent {
			record[k] = val
		}
		for k, val := range sm {
			if val != nil {
				record[k] = val
			} else if _, exists := record[k]; !exists {
				record[k] = nil
			}
		}
		out = append(out, record)
	}
	return out
}

// fieldTree represents a tree of field paths. An empty child map marks a leaf
// (the whole subtree at that key is kept verbatim). Nested keys narrow further.
//
// Examples:
//
//	["id", "name"]                   → {id: {}, name: {}}
//	["category_groups.categories.name"] → {category_groups: {categories: {name: {}}}}
//	["a.b", "a.c"]                   → {a: {b: {}, c: {}}}
//
// Arrays are implicit: a path descends through arrays transparently, applying
// to each element.
type fieldTree map[string]fieldTree

func parseFields(fields []string) fieldTree {
	root := fieldTree{}
	for _, f := range fields {
		parts := strings.Split(f, ".")
		node := root
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			child, ok := node[p]
			if !ok {
				child = fieldTree{}
				node[p] = child
			}
			node = child
		}
	}
	return root
}

func filterFields(v interface{}, fields []string) interface{} {
	return filterValue(v, parseFields(fields))
}

func filterValue(v interface{}, tree fieldTree) interface{} {
	// Empty tree = leaf: keep the value wholesale.
	if len(tree) == 0 {
		return v
	}
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, child := range val {
			if sub, ok := tree[k]; ok {
				result[k] = filterValue(child, sub)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, elem := range val {
			result[i] = filterValue(elem, tree)
		}
		return result
	default:
		return v
	}
}
