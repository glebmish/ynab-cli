package validate

import (
	"testing"
)

func TestPathParamValid(t *testing.T) {
	cases := []string{"i12345", "abc-def", "123", "some_id"}
	for _, c := range cases {
		if err := PathParam("id", c); err != nil {
			t.Errorf("PathParam(%q) unexpected error: %v", c, err)
		}
	}
}

func TestPathParamRejectsTraversal(t *testing.T) {
	cases := []string{"../etc/passwd", "foo/../bar", ".."}
	for _, c := range cases {
		if err := PathParam("id", c); err == nil {
			t.Errorf("PathParam(%q) expected error, got nil", c)
		}
	}
}

func TestPathParamRejectsControlChars(t *testing.T) {
	cases := []string{"ab\x00cd", "foo\nbar", "foo\rbar"}
	for _, c := range cases {
		if err := PathParam("id", c); err == nil {
			t.Errorf("PathParam(%q) expected error, got nil", c)
		}
	}
}

func TestPathParamRejectsQueryInjection(t *testing.T) {
	cases := []string{"id?fields=name", "id#fragment", "id&extra=1"}
	for _, c := range cases {
		if err := PathParam("id", c); err == nil {
			t.Errorf("PathParam(%q) expected error, got nil", c)
		}
	}
}

func TestPathParamRejectsPercentEncoding(t *testing.T) {
	cases := []string{"%2e%2e", "foo%2fbar", "%00"}
	for _, c := range cases {
		if err := PathParam("id", c); err == nil {
			t.Errorf("PathParam(%q) expected error, got nil", c)
		}
	}
}

func TestPathParamRejectsEmpty(t *testing.T) {
	if err := PathParam("id", ""); err == nil {
		t.Error("PathParam(\"\") expected error, got nil")
	}
}

func TestDateParamValid(t *testing.T) {
	cases := []string{"2026-04-11", "2025-01-01", "2030-12-31"}
	for _, c := range cases {
		if err := DateParam("date", c); err != nil {
			t.Errorf("DateParam(%q) unexpected error: %v", c, err)
		}
	}
}

func TestDateParamInvalid(t *testing.T) {
	cases := []string{"2026-13-01", "2026-04-32", "not-a-date", "04-11-2026", "20260411"}
	for _, c := range cases {
		if err := DateParam("date", c); err == nil {
			t.Errorf("DateParam(%q) expected error, got nil", c)
		}
	}
}

func TestJSONBodyValid(t *testing.T) {
	cases := []string{`{"name": "test"}`, `[1,2,3]`, `"hello"`, `null`}
	for _, c := range cases {
		if err := JSONBody(c); err != nil {
			t.Errorf("JSONBody(%q) unexpected error: %v", c, err)
		}
	}
}

func TestJSONBodyRejectsInvalid(t *testing.T) {
	cases := []string{`{not json}`, ``, `{"key": "val",}`}
	for _, c := range cases {
		if err := JSONBody(c); err == nil {
			t.Errorf("JSONBody(%q) expected error, got nil", c)
		}
	}
}

func TestJSONBodyRejectsControlChars(t *testing.T) {
	cases := []string{"{\"key\": \"val\x00ue\"}", "{\x00}"}
	for _, c := range cases {
		if err := JSONBody(c); err == nil {
			t.Errorf("JSONBody(%q) expected error, got nil", c)
		}
	}
}
