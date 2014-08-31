package pkgdep

import (
	"fmt"
	"github.com/serenitylinux/spack/libspack"
	"github.com/serenitylinux/spack/libspack/dep"
	"github.com/serenitylinux/spack/libspack/flag"
	"github.com/serenitylinux/spack/libspack/repo"
	"github.com/serenitylinux/spack/libspack/control"
	"github.com/serenitylinux/spack/libspack/pkginfo"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/cam72cam/go-lumberjack/color"
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

//Add rdeps if we are already installed
//We need to do this because changes to us might cause problems with packages that depend on us outside of the delta tree
//This links us into the existing "graph" on disk
func (pd *PkgDep) AddRdepConstraints(destdir string) {
	dep_info := pd.Repo.GetInstalledByName(pd.Name, destdir)
	if dep_info != nil {
		for _, rdep := range libspack.RdepList(dep_info.PkgInfo) {
			if pd.Graph == nil {
				log.Error.Println("WAT" + pd.Name)
			}
			
			
			//Copy pasta from depres
			depnode := pd.Graph.Find(rdep.Control.Name)
			//We are not part of the graph yet
			if depnode == nil {
				depnode = pd.Graph.Add(rdep.Control.Name, destdir)
				depnode.AddRdepConstraints(destdir)
			}
			
			
			var reason dep.Dep
			found := false
			
			curr_control := depnode.Control()
			curr_pkginfo := depnode.PkgInfo()
			
			all_flags := curr_pkginfo.ComputedFlagStates() //rdep.PkgInfo.ComputedFlagStates()
			all_deps := curr_control.ParsedDeps() //rdep.Control.ParsedDeps()
			
			for _, d := range all_deps.EnabledFromFlags(all_flags) {
				if d.Name == pd.Name {
					reason = d
					found = true
					break
				}
			}
			
			//This really should not happen
			if !found {
				log.Error.Printlnf("Unable to figure out why %s is a dep of %s, we may have reconstructed the on disk graph of packages incorrectly",
					pd.Name, depnode.Name)
				break
			}
			
			//Should return true, if not we have a serious problem with the packages on disk
			if !pd.AddParent(depnode, reason) {
				log.Error.Write([]byte("Conflicting package constraints on " + color.Red.String("already installed") +  " package " + pd.Name + ":" + "\n"))
				depnode.Constraints.PrintError("\t")
			}
		}
	}
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

func (pd *PkgDep) ValidFlags() bool {
	flagexpr := pd.Control().ParsedFlags()
	flagstates := pd.ComputedFlags()
	return flagexpr.Verify(flagstates)
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