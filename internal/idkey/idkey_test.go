package idkey

import "testing"

type stringerID struct{ v string }

func (s stringerID) String() string { return s.v }

type ptrStringerID struct{ v string }

func (s *ptrStringerID) String() string { return s.v }

func TestFormat(t *testing.T) {
	str := "abc"
	empty := ""
	num := 42

	tests := []struct {
		name  string
		input any
		want  string
		ok    bool
	}{
		{"nil", nil, "", false},
		{"string", "user-1", "user-1", true},
		{"empty string", "", "", false},
		{"string pointer", &str, "abc", true},
		{"empty string pointer", &empty, "", false},
		{"nil string pointer", (*string)(nil), "", false},
		{"int", 7, "7", true},
		{"int64", int64(-7), "-7", true},
		{"uint", uint(7), "7", true},
		{"int pointer", &num, "42", true},
		{"json number", float64(12000000), "12000000", true},
		{"json fractional number", 1.5, "1.5", true},
		{"value stringer", stringerID{v: "uuid-1"}, "uuid-1", true},
		{"pointer stringer", &ptrStringerID{v: "uuid-2"}, "uuid-2", true},
		{"nil pointer stringer", (*ptrStringerID)(nil), "", false},
		{"empty stringer", stringerID{}, "", false},
		{"unsupported", struct{ A int }{A: 1}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Format(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Format(%#v) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}
