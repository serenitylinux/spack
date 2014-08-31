package parser

import (
	"strings"
)

type Input struct {
	str string
	i int
}

func (in *Input) HasNext(length int) bool {
	return len(in.str) >= in.i + length
}
func (in *Input) Peek(length int) (string, bool) {
	if in.HasNext(length) {
		return in.str[in.i:in.i+length], true
	}
	return "", false
}
func (in *Input) Next(length int) (string, bool) {
	if in.HasNext(length) {
		in.i += length
		return in.str[in.i-length:in.i], true
	}
	return "", false
}
func (in *Input) ReadUntill(invalid string) (res string) {
	for c, exists := in.Peek(1); exists; c, exists = in.Peek(1) {
		if strings.Contains(invalid, c) {
			break
		}
		d, _ := in.Next(1)
		res += d
	}
	return res
}
func (in *Input) Rest() string {
	return in.str[in.i:]
}
func (in *Input) Index() int {
	return in.i
}
func (in *Input) IsNext(str string) bool {
	s, _ := in.Next(len(str))
	return s == str
}

func NewInput(s string) Input {
	return Input { s, 0 }
}