package argparse

import (
	"fmt"
	"os"
	"strings"
	"regexp"
	"strconv"
)


type Value interface {
	String() string
	Parse(string) error
	IsSet() bool
}

type StringValue struct {
	Value string
	isSet bool
}
func (s *StringValue) String() string { return string(s.Value) }
func (s *StringValue) IsSet () bool { return s.isSet }
func (s *StringValue) Get() string { return s.Value }
func (s *StringValue) Parse (value string) error {
	s.Value = value
	s.isSet = true
	return nil
}

type IntValue struct {
	Value int
	isSet bool
}
func (i *IntValue) String() string { return strconv.Itoa(int(i.Value)) }
func (i *IntValue) IsSet () bool { return i.isSet }
func (i *IntValue) Get() int { return i.Value }
func (i *IntValue) Parse (value string) error { 
	newval, e := strconv.Atoi(value)
	i.isSet = e == nil
	if i.isSet {
		i.Value = newval
	}
	return e
}

type BoolValue struct {
	Value bool
	isSet bool
}
func (b *BoolValue) String() string { return strconv.FormatBool(b.Value) }
func (b *BoolValue) IsSet () bool { return b.isSet }
func (b *BoolValue) Get() bool { return b.Value }
func (b *BoolValue) Parse (value string) error {
	newval, e := strconv.ParseBool(value)
	b.isSet = e == nil
	if (b.isSet) {
		b.Value = newval
	}
	return e
}

type Arg struct {
	Name string
	Usage string
	Value *Value
	DefValue interface{}
	Required bool
	IsBool bool
}

func RegisterString(argname string, defValue string, help string) *StringValue {
	value := &StringValue{defValue, false}
	RegisterArg(argname, value, defValue, help, false)
	return value
}

func RegisterInt(argname string, defValue int, help string) *IntValue {
	value := &IntValue{defValue, false}
	RegisterArg(argname, value, defValue, help, false)
	return value
}

func RegisterBool(argname string, defValue bool, help string) *BoolValue {
	value := &BoolValue{defValue, false}
	RegisterArg(argname, value, defValue, help, true)
	return value
}

var ArgList = make(map[string]Arg)

func RegisterArg(argname string, value Value, defValue interface{}, help string, booleanArg bool){
	if _, exists := ArgList[argname]; exists {
		panic("Argument already registered, " + argname)
	}
	ArgList[argname] = 
		Arg{
			argname,
			help,
			&value,
			defValue,
			false,
			booleanArg,
		}
}


func EvalDefaultArgs() []string {
	return EvalArgs(os.Args)
}

func HandleArg(argname string, value string) {
	arg,ok := ArgList[argname]
	if ok {
		error := (*arg.Value).Parse(value)
		if (error != nil) {
			fmt.Printf("Could not parse argument %s: ", argname)
			fmt.Println(error)
			Usage(2)
		}
	} else {
		fmt.Println("Unknown argument: " + argname)
		Usage(2)
	}
}

func EvalArgs(args []string) []string {
	matchHelp := regexp.MustCompile(`--help`)
	matchStdArg := regexp.MustCompile(`--(.)+=(.)+`)
	matchBooleanArg := regexp.MustCompile(`--(.)+`)
	
	leftover := []string{}
	
	for _, argString := range (args[1:]) {
		switch {
			case matchHelp.MatchString(argString):
				Usage(0)
			case matchStdArg.MatchString(argString):
				split := strings.SplitN(argString, "=", 2)
				argname := strings.TrimPrefix(split[0], "--")
				HandleArg(argname, split[1])
			case matchBooleanArg.MatchString(argString):
				argname := strings.TrimPrefix(argString, "--")
				
				arg,ok := ArgList[argname]
				if  !ok {
					fmt.Println("Unknown argument: " + argname)
					Usage(2)
				} else if arg.IsBool {
					HandleArg(argname, "true")
				} else {
					fmt.Println(fmt.Sprintf("--%s is not a boolean type, it requires an option", argname))
					Usage(2)
				}
			default:
				leftover = append(leftover,argString)
		}
	}
	
	return leftover
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a Arg) Len() int {
	return len(a.Pretty())
}

func (a Arg) Pretty() string {
	return fmt.Sprintf("--%s=%v", a.Name, a.DefValue)
}

var basename = os.Args[0]

func SetBasename(s string) {
	basename = s
}

func Usage(error int) {
	fmt.Println("Usage: ", basename)
	maxlen := 0
	
	for _, arg := range ArgList {
		maxlen = max(arg.Len(), maxlen)
	}
	
	fmt.Println()
	fmt.Println("Options:")
	for _, arg := range ArgList {
		spaces := strings.Repeat(" ", maxlen - arg.Len() + 3)
		fmt.Println("  ", arg.Pretty(), spaces, arg.Usage)
	}
	os.Exit(error)
}