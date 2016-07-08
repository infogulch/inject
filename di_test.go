package inject

import (
	"database/sql"
	"fmt"
	"testing"
)

type A int
type B int

func TestNew(t *testing.T) {
	di, err := New(A(0), B(1))
	if err != nil {
		t.Errorf("error creating injector: %v", err)
	}
	if n := *di.(*needle); len(n) != 2 {
		t.Errorf("needle is the wrong length: %v", n)
	}
	di, err = New(A(0), A(1))
	if err == nil {
		t.Errorf("new injector didn't catch duplicate types")
	}
}

func TestInject(t *testing.T) {
	di, _ := New(A(0), B(1), "foo", (*sql.DB)(nil))
	cases := []struct {
		name   string
		fn     interface{}
		expect string
		err    string
	}{
		{"AB", func(a A, b B) string { return fmt.Sprintf("%d %d", a, b) }, "0 1", ""},
		{"BA", func(b B, a A) string { return fmt.Sprintf("%d %d", a, b) }, "0 1", ""},
		{"strings", func(s string) string { return "a" + s + "b" }, "afoob", ""},
		{"missing type", func(i int) string { return "blah" }, "blah", "fn requires a type that's missing: int"},
		{"error ret", func(a A) (string, error) { return "blah", nil }, "blah", ""},
		{"non-error ret", func(a A) (string, int) { return "blah", 0 }, "blah", "cannot inject function with a non-error second return value: int"},
		{"sql", func(db *sql.DB) string { return "blah" }, "blah", ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			res, err := di.Inject(test.fn)
			if test.err != "" {
				if test.err != err.Error()[:len(test.err)] {
					t.Errorf("expected error, got %v", err)
				}
				if res != nil {
					t.Errorf("expected error, got result: %#+v", res)
				}
			} else if err != nil {
				t.Errorf("injection error: %v", err)
			} else if res.(string) != test.expect {
				t.Errorf(`injection failed: expected %q got %q`, test.expect, res.(string))
			}
		})
	}
}
