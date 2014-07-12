package pkgdep

import (
	"fmt"
	"libspack/dep"
	"libspack/repo"
	"libspack/control"
	"libspack/pkginfo"
)

/******************************************
Represents an installable package and it's rdeps
*******************************************/
type PkgDep struct {
	Control *control.Control
	Repo *repo.Repo
	FlagStates dep.FlagSet
	Dirty bool
	Happy bool
	ForgeOnly bool
	
	Parents PkgDepParentList
	
	Graph *PkgDepList
}

func New(c *control.Control, r *repo.Repo) *PkgDep {
	new := PkgDep { Control: c, Repo: r, Dirty: true, ForgeOnly: false }
	new.Parents = make(PkgDepParentList, 0)
	
	return &new
}
func (pd *PkgDep) String() string {
	return fmt.Sprintf("%s::%s%s", pd.Repo.Name, pd.Control.UUID(), pd.FlagStates)
}
func (pd *PkgDep) Equals(opd *PkgDep) bool {
	//TODO IsSubSet may not work in all cases
	return pd.Control.UUID() == opd.Control.UUID() && pd.FlagStates.IsSubSet(opd.FlagStates)
}
func (pd *PkgDep) MakeParentProud(opd *PkgDep, dep dep.Dep, isbdep bool) bool {
	if !pd.Parents.Contains(opd, isbdep) {
		//We need to add parent's requirements
		for _, pflag := range *dep.Flags {
			if _, exists := pd.FlagStates.Contains(pflag.Name); !exists {
				pd.FlagStates = append(pd.FlagStates, pflag)
			}
		}
		
		pd.Parents.Append(opd, dep, isbdep)
		pd.Dirty = true
	}
	return pd.SatisfiesParents()
}
func (pd *PkgDep) SatisfiesParents() bool {
	for _, p := range pd.Parents {
		if !p.IsProudOf(pd) {
			return false
		}
	}
	
	return true
}

func (pd *PkgDep) PkgInfo() *pkginfo.PkgInfo {
	p := pkginfo.FromControl(pd.Control)
	for _, flag := range pd.FlagStates {
		p.Flags = append(p.Flags, flag.String())
	}
	return p
}

func (pd *PkgDep) SpakgExists() bool {
	return pd.Repo.HasSpakg(pd.PkgInfo())
}

func (pd *PkgDep) IsInstalled(destdir string) bool {
	return pd.Repo.IsInstalled(pd.PkgInfo(), destdir)
}

func (pd *PkgDep) Find(name string) *PkgDep {
	return pd.Graph.Find(name)
}