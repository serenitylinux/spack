package pkgset

import (
	"libspack/pkginfo"
	"libspack/control"
)
import json "libspack/jsonhelper"

type PkgSet struct {
	Control control.Control
	PkgInfo pkginfo.PkgInfo
}
func (p *PkgSet) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}
func FromFile(filename string) (p *PkgSet, err error){
	var i PkgSet
	err = json.DecodeFile(filename, &i)
	if err == nil {
		*p = i
	}
	return
}
