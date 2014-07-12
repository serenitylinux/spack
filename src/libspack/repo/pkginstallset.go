package repo

import (
	"libspack/hash"
	"libspack/pkginfo"
	"libspack/control"
)
import json "libspack/jsonhelper"

type PkgInstallSet struct {
	Control *control.Control
	PkgInfo *pkginfo.PkgInfo
	Hashes  hash.HashList
}

func NewPkgIS(c *control.Control, p *pkginfo.PkgInfo, hash hash.HashList) *PkgInstallSet {
	return &PkgInstallSet{ c, p, hash };
}
func (p *PkgInstallSet) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}
func PkgISFromFile(filename string) (p *PkgInstallSet, err error) {
	var i PkgInstallSet
	err = json.DecodeFile(filename, &i)
	if err == nil {
		p = &i
	}
	return
}

func (repo *Repo) installSetFile(p pkginfo.PkgInfo, basedir string) string {
	return basedir + repo.installedPkgsDir() + p.UUID() + ".pkgset"
}