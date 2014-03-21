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

FlagList:
(FlagSpec,FlagSpec, ...)

*/

import (	
	"errors"
	"libspack/parser"
)

type Dep struct {
	condition *FlagSpec
	name string
	version1 *Version
	version2 *Version
	flags *FlagSet
}

const (
	GT = 1
	LT
	EQ
)

type Version struct {
	typ int
	ver string
}

type FlagSet struct {
	list []FlagSpec
}

type FlagSpec struct {
	enabled bool
	name string
}

func conditionPeek(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == "["
}

func versionPeek(in *parser.Input) bool {
	s, _ := in.Peek(1)
	return s == ">" || s == "<" || s == "="
}


func (d *Dep) parse(in *parser.Input) error {
	if conditionPeek(in) {
		in.Next(1)
		var new FlagSpec
		
		err := new.parse(in)
		if err != nil { return err }
		
		d.condition = &new
		
		if !in.IsNext("]") {
			return errors.New("Expected ']' at end of condition")
		}
	}
	
	d.name = in.ReadUntill("<>=()")
	if len(d.name) == 0 {
		return errors.New("Must specify dep package name")
	}
	
	if versionPeek(in) {
		var new Version
		err := new.parse(in)
		if err != nil { return err }
		d.version1 = &new
	}
	
	if versionPeek(in) && d.version1.typ != EQ {
		var new Version
		err := new.parse(in)
		if err != nil { return err }
		d.version2 = &new
	}
	
	//no requirements
	if !in.HasNext(1) {
		return nil
	}
	
	var new FlagSet
	err := new.parse(in)
	if err != nil {
		return err
	}
	d.flags = &new
	
	if in.HasNext(1) {
		return errors.New("Finished parsing, trailing chars '" + in.Rest() + "'")
	}
	
	return nil
}

func (s *FlagSet) parse(in *parser.Input) error {	
	if !in.IsNext("(") {
		return errors.New("Expected '(' to start flag set")
	}
	
	s.list = make([]FlagSpec, 0)
	
	for {
		var flag FlagSpec
		err := flag.parse(in)
		if err != nil { return err }
		
		s.list = append(s.list, flag)
		
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

func (f *FlagSpec) parse(in *parser.Input) error {
	sign, exists := in.Next(1)
	if !exists {
		return errors.New("Flag: Reached end of string while looking for sign")
	}
	
	f.enabled = "+" == sign
	
	f.name = in.ReadUntill("]")
	
	if len(f.name) == 0 {
		return errors.New("Flag: Nothing available after sign")
	}
	
	return nil
}