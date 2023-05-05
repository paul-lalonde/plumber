package plumb

type Msg struct {
	Src, Dst, Wdir, Typ string
	Attr []Attr
	Ndata int
	Data string 
}

type Attr struct {
	Name, Value string
}

func Packattr(attr []Attr) string{
	panic("unimplemented")
}