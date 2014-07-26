package flag

/*

-dev([+qt && -gtk] || [-qt && +gtk])
[is_enabled_default]name(deps)

exprlist  = expr + exprlist'
exprlist' = arg + exprlist || \0

expr = sub || flag
arg = '&&,||'

sub = '[' + exprlist + ']'
flag = '[+,-]s*'

*/

import (
	"errors"
	"strings"
	"libspack/parser"
	"lumberjack/color"
)

type Flag struct {
	Name string
	Enabled bool
}
func (f *Flag) Sign() string {
	sign := "+"
	if !f.Enabled {
		sign = "-"
	}
	return sign
}
func (f *Flag) String() string {
	return f.Sign() + f.Name
}
func (f *Flag) ColorString() string {
	if f.Enabled {
		return color.Green.String(f.String())
	} else {
		return color.Red.String(f.String())
	}
}

func Parse(in *parser.Input) (*Flag, error) {
	f := new(Flag)
	sign, exists := in.Next(1)
	if !exists {
		return nil, errors.New("Flag: Reached end of string while looking for sign")
	}
	
	f.Enabled = "+" == sign
	
	f.Name = in.ReadUntill("[]+-&|(),")
	
	if len(f.Name) == 0 {
		return nil, errors.New("Flag: Nothing available after sign")
	}
	
	return f, nil
}
func FlagFromString(s string) (*Flag, error) {
	s = strings.Replace(s, " ", "", -1)
	in := parser.NewInput(s)
	
	f, err := Parse(&in)
	if err != nil {
		return nil, err
	}
	return f, nil
}