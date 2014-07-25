package pkgdep

import (
	"strings"
	"libspack/dep"
	"libspack/flag"
	"lumberjack/log"
)

type Constraint struct {
	Parent *PkgDep
	dep dep.Dep
	reason string
}
func (c *Constraint) String() string {
	if c.Parent == nil {
		return c.reason
	} else {
		return c.Parent.String()
	}
}

type ConstraintList []Constraint

func (l *ConstraintList) Contains(p *PkgDep) bool {
	for _, n := range *l {
		if p == n.Parent{
			return true
		}
	}
	return false
}

func (l *ConstraintList) AppendParent(p *PkgDep, deps dep.Dep) {
	*l = append(*l, Constraint { p, deps, "Required by " + p.Name });
}

func (l *ConstraintList) AppendOther(reason string, deps dep.Dep) {
	*l = append(*l, Constraint { nil, deps, reason });
}

func (l *ConstraintList) ComputedFlags(p *PkgDep) (*flag.FlagList) {
	defaultf := p.Control().DefaultFlags();
	newlist := make(flag.FlagList, 0)
	
	//Sum up all of the constraints
	for _, c := range *l {
		if c.dep.Flags != nil {
			for _, f := range *c.dep.Flags {
				if currflag, exists := newlist.Contains(f.Name); exists {
					if currflag.Enabled != f.Enabled {
						//TODO ERRORORORORORO
						//We have a conflict!
						return nil
					}
				} else {
					newlist.Append(f)
				}
			}
		}
	}
	
	//merge constraints with defaults
	for _, deffl := range defaultf {
		if _, exists := newlist.Contains(deffl.Name); !exists {
			//We need to add the default to newlist
			newlist.Append(deffl)
		}
	}
	
	return &newlist
}

type VersionChecker func(string) bool

func (l *ConstraintList) ComputedVersionChecker() VersionChecker {
	versions := make([]*dep.Version,0)
	
	for _, c := range *l {
		if c.dep.Version1 != nil {
			versions = append(versions, c.dep.Version1)
		}
		if c.dep.Version2 != nil {
			versions = append(versions, c.dep.Version2)
		}
	}
	
	return func(str string) bool {
		for _, v := range versions {
			if ! v.Accepts(str) {
				return false
			}
		}
		
		return true
	}
}

func (l *ConstraintList) RemoveByParent(parent *PkgDep) bool {
	ret := false
	newl := make(ConstraintList, 0)
	for _, constraint := range *l {
		if constraint.Parent == parent {
			ret = true
		} else {
			newl = append(newl, constraint)
		}
	}
	*l = newl
	return ret
}

func (l *ConstraintList) PrintError(prefix string) {
	max := 0
	for _, c := range *l {
		ln := len(c.String())
		if ln > max {
			max = ln
		}
	}
	
	for _, c := range *l {
		str := prefix + c.String() + ":" + strings.Repeat(" ", max - len(c.String())) + "  " + c.dep.String() + "\n"
		log.Error.Write([]byte(str))
	}
}