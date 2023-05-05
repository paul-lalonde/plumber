package main

import (
	"os"
	"strings"
	"testing"

	"github.com/paul-lalonde/plumb"
)

func TestExpand(t *testing.T) {
	ttab := []struct {
		name, in, out string
	}{
		{"double quote", "quote '' quote", "quote  quote"},
		{"quoted quote", "quote ' '' ' quote", "quote  '  quote"},
		{"untransformed", "asdf", "asdf"},
		{"src", "$src", "source"},
		{"numeric expansion", "test $0 bar", "test amatch bar"},
		{"simple quote", "simple 'qu$src' quote", "simple qu$src quote"},
		{"end of line quote", "simple 'qu$src'", "simple qu$src"},
	}

	e := &Exec{
		msg: &plumb.Msg{
			Src:   "source",
			Dst:   "destination",
			Wdir:  "/tmp",
			Typ:   "type",
			Attr:  []plumb.Attr{{"name", "value"}},
			Ndata: 5,
			Data:  "hello",
		},
		match: [10]string{"amatch", "", "", "", "", "", "", "", "", ""},
		file:  "filename",
		dir:   "/tmp",
	}
	for _, tc := range ttab {

		got := expand(e, []rune(tc.in))
		if got != tc.out {
			t.Errorf("%s: expected '%v', got '%v'", tc.name, tc.out, got)
		}
	}
}

var fileaddr = `addrelem='((#?[0-9]+)|(/[A-Za-z0-9_\^]+/?)|[.$])'
addr=:($addrelem([,;+\-]$addrelem)*)

twocolonaddr = ([0-9]+)[:.]([0-9]+)
`

func TestReadRules(t *testing.T) {
	r := strings.NewReader(fileaddr)

	rules := newRules()
	err := rules.readrules("fileaddr", r)
	if err != nil {
		t.Errorf("Failed to read fileaddr: %v", err)
	}
	if len(rules) > 0 {
		t.Errorf("fileaddr should not have made any rules")
	}

	bf, err := os.Open("../testdata/basic")
	if err != nil {
		t.Fatalf("Failed to open test data")
	}
	rules = newRules()
	err = rules.readrules("basic", bf)
	if err != nil {
		t.Errorf("Failed to read fileaddr: %v", err)
	}
	if len(rules) < 1 {
		t.Errorf("basic should declare some rules")
	}

}
