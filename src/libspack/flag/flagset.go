package flag

import (
	"errors"
	"strings"
	"libspack/parser"
)

type FlagSet struct {
	Flag Flag
	list *exprlist
}

func FromString(s string) (fs FlagSet, err error) {
	s = strings.Replace(s, " ", "", -1)
	in := parser.NewInput(s)
	
	var f *Flag
	f, err = Parse(&in)
	if err != nil { return }
	fs.Flag = *f
	
	if exists := in.HasNext(1); !exists {
		//No conditions for flag
		return
	}
	
	if s, _ := in.Next(1); s != "(" {
		err = errors.New("Missing '(' after flag")
		return
	}
	
	var l *exprlist
	l, err = parseExprList(&in)
	if err != nil { return }
	fs.list = l;
	
	if s, _ := in.Next(1); s != ")" {
		err = errors.New("Missing ')' at the end of input")
		return
	}
	
	if exists := in.HasNext(1); exists {
		err = errors.New("Trailing chars after end of flag definition: '" + in.Rest() + "'")
		return
	}
	return
}

func (f FlagSet) Verify(list *FlagList) bool {
	if list.IsEnabled(f.Flag.Name) {
		return f.list.verify(list)
	}
	
	return true
}

func (f FlagSet) String() string {
	return f.Flag.String() + "(" + f.list.String() + ")"
}