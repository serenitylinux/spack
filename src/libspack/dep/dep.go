package dep

/*

[condition] name      versionspec        (depends)
[+-flag]  pkgname<>=version<>=version(+flag -flag)

FlagSpec:
+name
-name

ConditionFlag:
[FlagSpec]

PkgName:
all except "<>=("

Version:
>=version (multiple possible)
<=version (multiple possible)
==version (singular)

FlagSet:
(FlagSpec,FlagSpec, ...)

*/

import (
	"fmt"
	"strings"
	"errors"
	"libspack/parser"
	"libspack/flag"
)

type Dep struct {
	Condition *flag.Flag
	Name string
	Version1 *Version
	Version2 *Version
	Flags *flag.FlagList
}

func (d *Dep) String() string {
	return d.Name + d.Version1.String() + d.Version2.String() + "(" + d.Flags.String() + ")"
}

const (
	GT = 1
	LT = 2
	EQ = 3
)

type Version struct {
	typ int
	ver string
}
func (v *Version) String() string {
	s := ""
	if v == nil {
		return s
	}
	
	switch v.typ {
		case GT:
			s = ">"
		case LT:
			s = "<"
		case EQ:
			s = "="
	}
	return s + v.ver
}
func (v *Version) Accepts(verstr string) bool {
	switch v.typ {
		case GT:
			return verstr > v.ver
		case LT:
			return verstr < v.ver
		case EQ:
			return verstr == v.ver
	}
	panic(errors.New(fmt.Sprintf("Invalid version value: %d", v.typ)))
}

func conditionPeek(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == "["
}

func versionPeek(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == ">" || s == "<" || s == "="
}

func Parse(s string) (Dep, error) {
	s = strings.Replace(s, " ", "", -1)
	in := parser.NewInput(s)
	var d Dep
	err := d.parse(&in)
	return d, err
}

func (d *Dep) parse(in *parser.Input) error {
	if conditionPeek(in) {
		in.Next(1)
		var new flag.Flag
		
		err := new.Parse(in)
		if err != nil { return err }
		
		d.Condition = &new
		
		if !in.IsNext("]") {
			return errors.New("Expected ']' at end of condition")
		}
	}
	
	d.Name = in.ReadUntill("<>=()")
	if len(d.Name) == 0 {
		return errors.New("Must specify dep package name")
	}
	
	if versionPeek(in) {
		var new Version
		err := new.parse(in)
		if err != nil { return err }
		d.Version1 = &new
	}
	
	if versionPeek(in) && d.Version1.typ != EQ {
		var new Version
		err := new.parse(in)
		if err != nil { return err }
		d.Version2 = &new
	}
	
	//no requirements
	if !in.HasNext(1) {
		return nil
	}
	
	new := make(flag.FlagList, 0)
	err := parseFlagSet(&new, in)
	if err != nil {
		return err
	}
	d.Flags = &new
	
	if in.HasNext(1) {
		return errors.New("Finished parsing, trailing chars '" + in.Rest() + "'")
	}
	
	return nil
}

func parseFlagSet(s *flag.FlagList, in *parser.Input) error {	
	if !in.IsNext("(") {
		return errors.New("Expected '(' to start flag set")
	}
	
	for {
		var flag flag.Flag
		err := flag.Parse(in)
		if err != nil { return err }
		
		*s = append(*s, flag)
		
		str, _ := in.Next(1)
		if str != "," {
			//We are at the end
			
			if str != ")" {
				return errors.New("Invalid char '" + str + "', expected ')'")
			}
			
			break;
		}
	}
	return nil
}

func (v *Version) parse(in *parser.Input) error {
	s, _ := in.Next(2)
	switch s {
		case ">=": v.typ = GT
		case "<=": v.typ = LT
		case "==": v.typ = EQ
		default:   return errors.New("Invalid condition '" + s + "', expected [<>=]=")
	}
	v.ver = in.ReadUntill("<>=(")
	if len(v.ver) == 0 {
		return errors.New("[<>=]= must be followed by a version")
	}
	return nil
}


type DepList []Dep
func (list *DepList) EnabledFromFlags(fs flag.FlagList) DepList {
	res := make(DepList, 0)
	for _, dep := range *list {
		//We have no include condition
		if dep.Condition == nil {
			res = append(res, dep)
			continue
		}
		
		for _, flag := range fs {
			if *dep.Condition == flag {
				res = append(res, dep)
				break;
			}
		}
	}
	return res
}