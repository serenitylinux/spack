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
	"strings"
	"errors"
	"libspack/parser"
	"libspack/flag"
)

type Dep struct {
	Condition *flag.Flag
	Name string
	version1 *Version
	version2 *Version
	Flags *FlagSet
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

type FlagSet []flag.Flag

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
	
	new := make(FlagSet, 0)
	err := new.parse(in)
	if err != nil {
		return err
	}
	d.Flags = &new
	
	if in.HasNext(1) {
		return errors.New("Finished parsing, trailing chars '" + in.Rest() + "'")
	}
	
	return nil
}

func (s *FlagSet) parse(in *parser.Input) error {	
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


func (l *FlagSet) String() string {
	str := ""
	for _, flag := range *l {
		str += flag.String() + " "
	}
	return str
}
func (l *FlagSet) IsSubSet(ol FlagSet) bool {
	for _, flag := range *l {
		found := false
		for _, oflag := range ol {
			if oflag.Name == flag.Name {
				found = oflag.Enabled == flag.Enabled
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (l *FlagSet) Contains(f string) (*flag.Flag, bool) {
	for _, flag := range *l {
		if flag.Name == f {
			return &flag, true
		}
	}
	return nil, false
}