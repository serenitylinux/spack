package control

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"io"
	"bytes"
	"libspack/log"
	"libspack/misc"
	"libspack/flag"
	"libspack/dep"
	"libspack/helpers/json"
)

type Control struct {
	Name string
	Version string
	Iteration int
	Description string
	Url string
	Src []string
	Arch []string
	
	Bdeps []string
	Deps []string
	Flags []string
	
	parsedFlags []flag.FlagSet
	parsedDeps dep.DepList
	parsedBDeps dep.DepList
	
	//Provides (libjpeg, cc)
	//Provides Hook (update mime types)
}

type ControlList []Control

func (c *Control) String() string {
	return json.Stringify(c)
}

func (c *Control) UUID() string {
	return fmt.Sprintf("%s-%s%%%d", c.Name, c.Version, c.Iteration)
}

func (c ControlList) String() string {
	return json.Stringify(c)
}


func (c *Control) ToFile(filename string) error {
	return json.EncodeFile(filename, true, c)
}

func (c *ControlList) ToFile(filename string) error {
	return json.EncodeFile(filename, true, c)
}


func FromFile(filename string) (*Control, error) {
	var c Control
	err := json.DecodeFile(filename, &c)
	return &c, err
}

func FromReader(reader io.Reader) (*Control, error) {
	var c Control
	err := json.DecodeReader(reader, &c)
	return &c, err
}

//TODO consolidate into a single function

func fromTemplateString(template string) (*Control, error) {
		commands := `
%s

function lister() {
	local set i
	set=""
	for i in "$@"; do
		echo -en "$set\"$i\""
		set=", "
	done
}

srcval="$(lister ${src[@]})"
bdepsval="$(lister ${bdeps[@]})"
depsval="$(lister ${deps[@]})"
archval="$(lister ${arch[@]})"
flagsval="$(lister ${flags[@]})"

cat << EOT
{
  "Name": "$name",
  "Version": "$version",
  "Iteration": $iteration,
  "Description": "$desc",
  "Url": "$url",
  "Src": [$srcval],
  "Bdeps": [ $bdepsval ],
  "Deps": [ $depsval ],
  "Arch": [ $archval ],
  "Flags": [ $flagsval ]
},
EOT`
	commands = fmt.Sprintf(commands, template)
	
	var buf bytes.Buffer
	cmd := exec.Command("bash", "-ec", commands)
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil { return nil, err	}
	return FromReader(bytes.NewReader(buf.Bytes()))
}

func FromTemplateFile(template string) (*Control, error) {
	var str string
	err := misc.WithFileReader(template, func (r io.Reader) {
		str = misc.ReaderToString(r)
	})
	if err != nil {
		return nil, err
	}
	
	//Don't care if does not exist
	misc.WithFileReader(filepath.Dir(template) + "/default", func (r io.Reader) {
		str += misc.ReaderToString(r)
	})
	
	return fromTemplateString(str)
}



func (c *Control) ParsedFlags() []flag.FlagSet {
	if c.parsedFlags == nil {
		c.parsedFlags = make([]flag.FlagSet, 0)
		for _, s := range c.Flags {
			flag, err := flag.FromString(s)
			if err != nil {
				log.WarnFormat("Invalid flag in package %s '%s': %s", c.Name, s, err)
				continue
			}
			c.parsedFlags = append(c.parsedFlags, flag)
		}
	}
	return c.parsedFlags
}
func (c *Control) DefaultFlags() flag.FlagList {
	res := make(flag.FlagList, 0)
	for _, fs := range c.ParsedFlags() {
		res = append(res, fs.Flag)
	}
	return res
}

func (c *Control) ParsedDeps() dep.DepList {
	if c.parsedDeps == nil {
		c.parsedDeps = make(dep.DepList, 0)
		for _, s := range c.Deps {
			dep, err := dep.Parse(s)
			if err != nil {
				log.WarnFormat("Invalid dep in package %s '%s': %s", c.Name, s, err)
				continue
			}
			c.parsedDeps = append(c.parsedDeps, dep)
		}
	}
	return c.parsedDeps
}

func (c *Control) ParsedBDeps() dep.DepList {
	if c.parsedBDeps == nil {
		c.parsedBDeps = make(dep.DepList, 0)
		for _, s := range c.Bdeps {
			dep, err := dep.Parse(s)
			if err != nil {
				log.WarnFormat("Invalid Bdep in package %s '%s': %s", c.Name, s, err)
				continue
			}
			c.parsedBDeps = append(c.parsedBDeps, dep)
		}
	}
	return c.parsedBDeps
}
