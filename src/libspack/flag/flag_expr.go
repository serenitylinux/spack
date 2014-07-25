package flag

import (
	"errors"
	"libspack/parser"
)

type expr struct {
	list *exprlist
	flag *Flag
}

func parseExpr(in *parser.Input) (*expr, error) {
	e := new(expr)
	if s,_ := in.Peek(1); s == "[" {
		in.Next(1)
		
		newl, err := parseExprList(in)
		if err != nil {
			return nil, err
		}
		
		if newl == nil {
			return nil, errors.New("[ ... ] must contain at least one flag")
		}
		
		e.list = newl
		
		s,_ := in.Next(1)
		if s != "]" {
			return nil, errors.New("Expression: Unexpected char '" + s + "'")
		}
	} else {
		newf, err := Parse(in)
		if err != nil {
			return nil, err
		}
		
		e.flag = newf
	}
	return e, nil
}
func (e *expr) verify(flist *FlagList) bool {
	if e.list != nil {
		return e.list.verify(flist)
	} else {
		return e.flag.Enabled == flist.IsEnabled(e.flag.Name)
	}
}
func (e *expr) String() string {
	if e.list != nil {
		return "[" + e.list.String() + "]"
	} else {
		return e.flag.String()
	}
}
