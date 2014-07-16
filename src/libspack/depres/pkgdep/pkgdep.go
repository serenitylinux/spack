package pkgdep

import (
	"fmt"
	"libspack/dep"
	"libspack/flag"
	"libspack/repo"
	"libspack/control"
	"libspack/pkginfo"
)

/******************************************
Represents an installable package and it's rdeps
*******************************************/
type PkgDep struct {
	Name string
//	Control *control.Control //is tied to a version  should be computed
	Repo *repo.Repo
//	FlagStates dep.FlagSet //should be computed
	Dirty bool
	IsReinstall bool
	ForgeOnly bool
	
	Constraints ConstraintList
	
	Graph *PkgDepList
}

func New(name string, r *repo.Repo) *PkgDep {
	new_pd := PkgDep { Name: name, Repo: r, Dirty: true, ForgeOnly: false }
	new_pd.Constraints = make(ConstraintList, 0)
	
	return &new_pd
}
func (pd *PkgDep) String() string {
	return fmt.Sprintf("%s::%s(%s)", pd.Repo.Name, pd.Control().UUID(), pd.ComputedFlags())
}

//note: old parents should be removed, so we should never need to modify an existing constraint
func (pd *PkgDep) AddParent(parent *PkgDep, reason dep.Dep) bool {
	if !pd.Constraints.Contains(parent) {
		pd.Constraints.AppendParent(parent, reason)
		pd.Dirty = true
	}
	return pd.Exists()
}
func (pd *PkgDep) RemoveParent(parent *PkgDep) bool {
	return pd.Constraints.RemoveByParent(parent)
}

func (pd *PkgDep) Exists() bool {
	return pd.Control() != nil && pd.ComputedFlags() != nil
}

func (pd *PkgDep) Control() *control.Control {
	return pd.Repo.GetPackageByVersionChecker(pd.Name, pd.Constraints.ComputedVersionChecker())
}

func (pd *PkgDep) PkgInfo() *pkginfo.PkgInfo {
	p := pkginfo.FromControl(pd.Control())
	flags := pd.ComputedFlags()
	
	if flags == nil { 
		return nil
	}
	
	for _, flag := range *flags {
		p.SetFlagState(&flag)
	}
	return p
}

func (pd *PkgDep) ComputedFlags() *flag.FlagList {
	return pd.Constraints.ComputedFlags(pd)
}

func (pd *PkgDep) SpakgExists() bool {
	return pd.Repo.HasSpakg(pd.PkgInfo())
}

func (pd *PkgDep) IsInstalled(destdir string) bool {
	return !pd.IsReinstall && pd.Repo.IsInstalled(pd.PkgInfo(), destdir)
}

func (pd *PkgDep) FindInGraph(name string) *PkgDep {
	return pd.Graph.Find(name)
}