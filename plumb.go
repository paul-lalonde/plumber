package plumb

import (
	"strings"
)

type Msg struct {
	Src, Dst, Wdir, Typ string
	Attr []Attr
	Ndata int
	Data string 
}

type Attr struct {
	Name, Value string
}

func unquote(s string) string {
	if !(s[0] == '\'' && s[len(s)-1] == '\'') {
		return s
	}
	return strings.Replace(s, "''", "'", -1)
}

// Only quote if needed
func quoteIfNeeded(s string) string {
	panic("unimplemented")
}
		
func Unpackattr(s string) Attr {
	attr := Attr{"",""}
	s = strings.TrimSpace(s)
	words := strings.SplitN(s, "=", 1)
	if len(words) > 0 {
		attr.Name = words[0]
	}
	if len(words) > 1 {
		attr.Value = unquote(words[1])
	}
	return attr
}

func Packattr(attr []Attr) string{
	panic("unimplemented")
}

func (m Msg)Lookup(s string) string {
	for _, a := range m.Attr {
		if a.Name == s {
			return a.Value
		}
	}
	return ""
}

func (m *Msg)Delattr(a string) bool {
	for i:=0; i < len(m.Attr); i++ {
		if m.Attr[i].Name == a {
			m.Attr = append(m.Attr[0:i], m.Attr[i+1:]...)
			return true
		}
	}
	return false
}