package control

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"io"
	"bytes"
	"libspack/misc"
)
import json "libspack/jsonhelper"

type Control struct {
	Name string
	Version string
	Iteration int
	Description string
	Url string
	Src []string
	Bdeps []string //TODO more complex object?
	Deps []string //TODO more complex object?
	Arch []string
	Flags []string
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
	for i in $@; do
		echo -en "$set    \"$i\""
		set=",\n"
	done
}

srcval="$(lister $src)"
bdepsval="$(lister $bdeps)"
depsval="$(lister $deps)"
archval="$(lister $arch)"
flagsval="$(lister $flags)"

cat << EOT
{
  "Name": "$name",
  "Version": "$version",
  "Iteration": $iteration,
  "Description": "$desc",
  "Url": "$url",
  "Src": [ $srcval ],
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
