package pkgdep

import (
	"libspack/dep"
)

/*****************************************
Struct that represents how a parent depends on a child
******************************************/
type PkgDepParent struct {
	Parent *PkgDep
	dep dep.Dep
	isbdep bool;
}
func (p * PkgDepParent) IsProudOf(pd *PkgDep) bool {
	return p.dep.Flags.IsSubSet(pd.FlagStates)
}

type PkgDepParentList []PkgDepParent

func (l *PkgDepParentList) Contains(p *PkgDep, isbdep bool) bool {
	for _, n := range *l {
		if p.Equals(n.Parent) && isbdep == n.isbdep {
			return true
		}
	}
	return false
}

func (l *PkgDepParentList) Append(p *PkgDep, deps dep.Dep, isbdep bool) {
	*l = append(*l, PkgDepParent { p, deps, isbdep });
}