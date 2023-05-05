package main

import (
	"regexp"

	"github.com/paul-lalonde/plumb"
)

type Object int

const (
	OArg = Object(iota)
	OAttr
	OData
	ODst
	OPlumb
	OSrc
	OType
	OWdir
)

type Verb int

const (
	VAdd = Verb(iota) /* apply to OAttr only */
	VClient
	VDelete /* apply to OAttr only */
	VIs
	VIsdir
	VIsfile
	VMatches
	VSet
	VStart
	VTo
)

type Rule struct {
	obj   Object
	verb  Verb
	arg   string /* unparsed string of all arguments */
	qarg  string /* quote-processed arg string */
	regex *regexp.Regexp
}

type Rules []*Ruleset
func newRules() Rules {
	return []*Ruleset{}
}

type Ruleset struct {
	pat  []Rule
	act  []Rule
	port string
}

type Exec struct {
	msg           *plumb.Msg
	match         [10]string
	text		string
	p0, p1        int
	clearclick    bool
	setdata       bool
	holdforclient bool
	file          string
	dir           string
}
