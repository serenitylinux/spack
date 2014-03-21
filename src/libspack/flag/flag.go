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
)

type op bool //true = and, false = or

type flag struct {
	name string
	enabled bool
}

type expr struct {
	list *el
	flag *flag
}

type el struct {
	e expr
	op *op
	next *el
}


func (f *flag) parse(in *parser.Input) error {
	sign, exists := in.Next(1)
	if !exists {
		return errors.New("Flag: Reached end of string while looking for sign")
	}
	
	f.enabled = "+" == sign
	
	f.name = in.ReadUntill("[]+-&|()")
	
	if len(f.name) == 0 {
		return errors.New("Flag: Nothing available after sign")
	}
	
	return nil
}

func (e *expr) parse(in *parser.Input) error {
	if s,_ := in.Peek(1); s == "[" {
		in.Next(1)
		
		new := el { }
		
		err := new.parse(in)
		if err != nil {
			return err
		}
		e.list = &new
		
		s,_ := in.Next(1)
		if s != "]" {
			return errors.New("Expression: Unexpected char '" + s + "'")
		}
	} else {
		new := flag { }
		
		err := new.parse(in)
		if err != nil {
			return err
		}
		
		e.flag = &new
	}
	return nil
}

func (op *op) parse(in *parser.Input) error {
	s, _ := in.Next(2)
	switch s {
		case "&&": *op = true
		case "||": *op = false
		default: return errors.New("Op: Invalid operation '"+ s +"'")
	}
	return nil;
}
func op_isnext(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == "&" || s == "|"
}

func (list *el) parse(in *parser.Input) error {
	err := list.e.parse(in)
	if err != nil {
		return err
	}
	
	if op_isnext(in) {
		var nop op
		var nel el
		
		if err := nop.parse(in); err != nil {
			return err
		}
		
		if err := nel.parse(in); err != nil {
			return err
		}
		
		list.op = &nop
		list.next = &nel
	}
	return nil
}

type FlagSet struct {
	flag flag
	list el
}

func FromString(s string) (fs FlagSet, err error) {
	s = strings.Replace(s, " ", "", -1)
	in := parser.NewInput(s)
	
	err = fs.flag.parse(&in)
	if err != nil { return }
	
	if exists := in.HasNext(1); !exists {
		//No conditions for flag
		return
	}
	
	if s, _ := in.Next(1); s != "(" {
		err = errors.New("Missing '(' after flag")
		return
	}
	
	err = fs.list.parse(&in)
	if err != nil { return }
	
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