package flag

import (
	"errors"
	"libspack/parser"
)

const (
	And = true
	Or = false
)
type op bool

func parseOp(in *parser.Input) (*op, error) {
	s, _ := in.Next(2)
	res := new(op)
	switch s {
		case "&&": *res = true
		case "||": *res = false
		default: return nil, errors.New("Op: Invalid operation '"+ s +"'")
	}
	return res, nil
}
func (op *op) String() string {
	if *op == And {
		return " && "
	} else {
		return " || "
	}
}
func op_isnext(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == "&" || s == "|"
}
