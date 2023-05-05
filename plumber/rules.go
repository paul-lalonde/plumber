package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/paul-lalonde/plumb"
)

type Input struct {
	file    string
	scanner *bufio.Scanner
	lineno  int
	next    *Input
}

var input *Input

type Var struct {
	name, value, qvalue string
}

var vars = map[string]Var{}

var badports = map[string]bool{
	".":    true,
	"..":   true,
	"send": true,
}

var objects = map[string]Object{
	"arg":   OArg,
	"attr":  OAttr,
	"data":  OData,
	"dst":   ODst,
	"plumb": OPlumb,
	"src":   OSrc,
	"type":  OType,
	"wdir":  OWdir,
}
var objectNames = []string{
	"arg",
	"attr",
	"data",
	"dst",
	"plumb",
	"src",
	"type",
	"wdir",
}

var verbs = map[string]Verb{
	"add":     VAdd,
	"client":  VClient,
	"delete":  VDelete,
	"is":      VIs,
	"isdir":   VIsdir,
	"isfile":  VIsfile,
	"matches": VMatches,
	"set":     VSet,
	"start":   VStart,
	"to":      VTo,
}

var verbNames = []string{
	"add",
	"client",
	"delete",
	"is",
	"isdir",
	"isfile",
	"matches",
	"set",
	"start",
	"to",
}

func printinputstackrev(in *Input) *ParseError {
	if in == nil {
		return nil
	}
	e := printinputstackrev(in)
	pe := in.NewError("\t at")
	pe.Err = e
	return pe
}

func (in *Input) pushinput(name string, fd io.Reader) {
	depth := 0
	for in := input; in != nil; in = in.next {
		if depth >= 10 {
			errorf("include stack too deep; max 10")
			panic("Recovery unimplemented")
		}
		depth++
	}
	oldin := *in
	*in = Input{file: name, next: &oldin, scanner: bufio.NewScanner(fd)}
}

func (in *Input) popinput() bool {
	if in.next == nil {
		return false
	}
	*in = *in.next
	return in.file != ""
}

func lookup(s string, tab []string) int {
	for i, t := range tab {
		if s == t {
			return i
		}
	}
	return -1
}

func variable(s string) string {
	if v, ok := vars[s]; ok {
		return v.qvalue
	} else {
		return ""
	}
}

func setvariable(s, val, qval string) {
	v := Var{name: s, value: val, qvalue: qval}
	vars[s] = v
}

func filename(e *Exec, name string) string {
	if name != "" {
		return filepath.Clean(name)
	}
	if filepath.IsAbs(string(e.msg.Data)) {
		return filepath.Clean(string(e.msg.Data))
	}
	path := filepath.Join(string(e.msg.Wdir), string(e.msg.Data))
	return filepath.Clean(path)
}

func dollar(e *Exec, s string) (ret string, consumed int) {
	if e != nil && s[0] >= '0' && s[0] <= '9' {
		return e.match[s[0]-'0'], 1
	}
	idx := strings.IndexFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) })
	if idx == 0 {
		idx = len(s)
	}
	varname := s[:idx]
	if e == nil {
		return variable(varname), idx
	}
	switch s {
	case "src":
		return e.msg.Src, idx
	case "dst":
		return e.msg.Dst, idx
	case "dir":
		return e.dir, idx
	case "attr":
		return plumb.Packattr(e.msg.Attr), idx
	case "data":
		return e.msg.Data, idx
	case "file":
		return e.file, idx
	case "type":
		return e.msg.Typ, idx
	case "wdir":
		return e.msg.Wdir, idx
	default:
		return variable(varname), idx
	}
	panic("Notreached")
}

func expand(e *Exec, s []rune) string {
	quoting := false

	out := strings.Builder{}

	for i := 0; i < len(s); i++ {
		r := s[i]
		if r == '\'' { // Toggle quoting, handling double-quote quote escape
			if !quoting {
				quoting = true
			} else if len(s) > i+1 && s[i+1] == '\'' {
				i++
				out.WriteRune('\'')
			} else {
				quoting = false
			}
			continue
		}
		if quoting || r != '$' {
			out.WriteRune(r)
			continue
		}
		// Variable expansion
		val, consumed := dollar(e, string(s[i+1:]))
		if val == "" {
			out.WriteRune('$')
			continue
		}
		i += consumed
		out.WriteString(val)
	}
	return out.String()
}

func (in *Input) parserule(r *Rule) (err error) {
	r.qarg = expand(nil, []rune(r.arg))
	switch r.obj {
	case OArg,
		OAttr,
		OData,
		ODst,
		OType,
		OWdir,
		OSrc:
		switch {
		case r.verb == VClient, r.verb == VStart, r.verb == VTo:
			return in.NewError(
				"%s  not  valid verb for object %s", verbNames[r.verb], objectNames[r.obj])
		case r.obj != OAttr && (r.verb == VAdd || r.verb == VDelete):
			return in.NewError(
				"%s not valid verb for object %s", verbNames[r.verb], objectNames[r.obj])
		case r.verb == VMatches:
			r.regex, err = regexp.Compile(r.qarg)
			return err
		}
	case OPlumb:
		if r.verb != VClient && r.verb != VStart && r.verb != VTo {
			return in.NewError(
				"%s not valid verb for object %s", verbNames[r.verb], objectNames[r.obj])
		}
	}
	return nil
}

func assignment(p []rune) bool {
	if !unicode.IsLetter(p[0]) {
		return false
	}
	n := 0
	for n = 0; n < len(p); n++ {
		if !(unicode.IsLetter(p[n]) || unicode.IsNumber(p[n])) {
			break
		}
	}
	varname := p[0:n]
	for p[n] == ' ' || p[n] == '\t' {
		n++
	}
	if p[n] != '=' {
		return false
	}
	n++
	for p[n] == ' ' || p[n] == '\t' {
		n++
	}
	qval := expand(nil, p[n:])
	setvariable(string(varname), string(p), qval)
	return true
}

func (in *Input) include(s string) (wasInclude bool, err error) {
	if !strings.HasPrefix(s, "include") {
		return false, nil
	}
	args := strings.Fields(s)
	if len(args) < 2 || args[0] != "include" || args[1][0] == '#' ||
		(len(args) > 2 && args[2][0] != '#') {
		return false, in.NewError("malformed include statement")
	}
	t := args[1]
	fp, err := os.Open(t)
	if err != nil && t[0] != '/' && t[0:2] != "./" && t[0:3] != "../" {
		// Try the plumbing directory
		t = unsharp(fmt.Sprintf("#9/plumb/%s", t))
	}
	if err != nil {
		return false, in.NewError("can't open %s for inclusion", t)
	}
	in.pushinput(t, fp)
	return true, nil
}

func (in *Input) readrule() (rp *Rule, err error) {
	rp = &Rule{}
Top:
	if in.scanner == nil || !in.scanner.Scan() {
		if !in.popinput() {
			return nil, io.EOF
		}
	}

	line := in.scanner.Text()
	in.lineno++
	line = strings.TrimSpace(line)
	line = strings.Split(line, "#")[0]

	if line == "" { /* empty or comment line */
		return nil, nil
	}

	wasInclude, err := in.include(line)
	if err != nil {
		return nil, err
	}
	if wasInclude {
		goto Top
	}

	if assignment([]rune(line)) {
		return nil, nil
	}

	words := strings.Fields(line)

	// Object
	if len(words) < 1 {
		return nil, in.NewError("malformed rule")
	}
	ok := false
	rp.obj, ok = objects[words[0]]
	if !ok {
		if words[0] == "kind" {
			rp.obj = OType
		} else {
			return nil, in.NewError("unknown object %s", words[0])
		}
	}

	// verb
	if len(words) < 2 {
		return nil, in.NewError("malformed rule")
	}
	rp.verb, ok = verbs[words[1]]
	if !ok {
		return nil, in.NewError("unknown verb %s", words[1])
	}

	// argument
	if len(words) < 3 {
		return nil, in.NewError("malformed rule")
	}
	rp.arg = words[2]

	err = in.parserule(rp)
	return rp, err
}

func NewRuleset() *Ruleset {
	rs := Ruleset{
		pat:  []Rule{},
		act:  []Rule{},
		port: "",
	}
	return &rs
}

func (in *Input) readruleset() (*Ruleset, error) {

	plan9root := unsharp("#9/")
	if plan9root != "#9/" {
		setvariable("plan9", plan9root, plan9root)
	}

	for {
		rs := NewRuleset()
		inrule := false
		ncmd := 0
		for {
			r, err := in.readrule()
			if err == io.EOF {
				break
			}
			if r == nil {
				if inrule {
					break
				}
				continue
			}
			inrule = true
			switch r.obj {
			case OArg, OAttr, OData, ODst, OType, OWdir, OSrc:
				rs.pat = append(rs.pat, *r)
			case OPlumb:
				rs.act = append(rs.act, *r)
				if r.verb == VTo {
					if len(rs.pat) > 0 && rs.port != "" { // npat==0 implies port declaration
						return nil, in.NewError("too many ports")
					}
					if _, ok := badports[r.qarg]; ok {
						return nil, in.NewError("illegal port name %s", r.qarg)
					}
					rs.port = r.qarg
				} else {
					ncmd++
				}
			}
		}

		if ncmd > 1 {
			return nil, in.NewError("ruleset has more than one client or start action")
		}
		if len(rs.pat) > 0 && len(rs.act) > 0 {
			return rs, nil
		}
		if len(rs.pat) == 0 && len(rs.act) == 0 {
			return nil, nil
		}
		if len(rs.act) == 0 || rs.port == "" {
			return nil, in.NewError("ruleset must have patterns and actions")
		}

		// declare ports
		for i := range rs.act {
			addport(rs.act[i].qarg)
		}

	}
}

func readrules(name string, fd io.Reader) ([]*Ruleset, error) {
	var in Input
	rules := []*Ruleset{}

	in.pushinput(name, fd)
	for {
		rs, err := in.readruleset()
		if err == io.EOF || rs == nil {
			return rules, nil
		}
		if err != nil {
			return nil, err
		}
		rules = append(rules, rs)
	}
	in.popinput()

	return rules, nil
}
