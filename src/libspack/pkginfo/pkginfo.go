package pkginfo

import (
	"fmt"
	"io"
	"time"
	"hash/crc32"
	"libspack/control"
	"libspack/flag"
	"libspack/log"
)

import json "libspack/jsonhelper"

type PkgInfo struct {
	Name string
	Version string
	Iteration int
	BuildDate time.Time
	Flags []string
	parsedFlags []flag.FlagSet
}

type PkgInfoList []PkgInfo

func (p *PkgInfo) String() string {
	return json.Stringify(p)
}
func (p *PkgInfo) UUID() string {
	return fmt.Sprintf("%s-%s%%%d::%x", p.Name, p.Version, p.Iteration, p.flagHash())
}

func (p *PkgInfo) flagHash() uint32 {
	str := p.Name
	for _, flag := range p.Flags {
		str += flag
	}
	return crc32.ChecksumIEEE([]byte(str))
}

func FromControl(c *control.Control) *PkgInfo {
	p := PkgInfo{ Name: c.Name, Version: c.Version, Flags: make([]string,0), Iteration: c.Iteration }
	return &p
}

func (p *PkgInfo) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}

func (p *PkgInfo) ParsedFlags() []flag.FlagSet {
	if p.parsedFlags == nil {
		p.parsedFlags = make([]flag.FlagSet, 0)
		for _, s := range p.Flags {
			flag, err := flag.FromString(s)
			if err != nil {
				log.WarnFormat("Invalid flag in package %s '%s': %s", p.Name, s, err)
				continue
			}
			p.parsedFlags = append(p.parsedFlags, flag)
		}
	}
	return p.parsedFlags
}

func FromFile(filename string) (*PkgInfo, error) {
	var i PkgInfo
	err := json.DecodeFile(filename, &i)
	return &i, err
}

func FromReader(reader io.Reader) (*PkgInfo, error) {
	var i PkgInfo
	err := json.DecodeReader(reader, &i)
	return &i, err
}
