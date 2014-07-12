package pkgdep

import (
	"fmt"
	"libspack"
	"libspack/log"
	"libspack/misc"
	"libspack/flagconfig"
)

/******************************************
List of Pkg dependencies
*******************************************/
type PkgDepList []*PkgDep

func (pdl *PkgDepList) Contains(pd *PkgDep) bool {
	for _, item := range *pdl {
		if item.Equals(pd) {
			return true
		}
	}
	return false
}

func (pdl *PkgDepList) Append(c *PkgDep) {
	*pdl = append(*pdl, c)
}
//http://blog.golang.org/slices Magics
func (pdl *PkgDepList) Prepend(c *PkgDep) {
	*pdl = (*pdl)[0 : len(*pdl)+1] //Increase size by 1
	copy((*pdl)[1:], (*pdl)[0:])   //shift array up by 1
	(*pdl)[0] = c                  //set new first element
}

func (pdl *PkgDepList) Print() {
	i := 0
	for _, item := range *pdl {
		str := item.String() + " "
		i += len(str)
		if i > misc.GetWidth()-10 {
			fmt.Println()
			i = len(str)
		}
		fmt.Print(str)
	}
	fmt.Println()
}

//http://stackoverflow.com/a/19239850
func (pdl *PkgDepList) Reverse() {
	for i, j := 0, len(*pdl)-1; i < j; i, j = i+1, j-1 {
		(*pdl)[i], (*pdl)[j] = (*pdl)[j], (*pdl)[i]
	}
}
func (pdl *PkgDepList) Add(dep string, destdir string) *PkgDep {
	//Create new pkgdep node
	ctrl, repo := libspack.GetPackageLatest(dep)
	if ctrl == nil {
		log.Error("Unable to find package ", dep)
		return nil
	}
	
	depnode := New(ctrl, repo)
	pdl.Append(depnode)
	
	//Add global flags to new depnode
	globalflags, exists := flagconfig.GetAll(destdir)[ctrl.Name]
	if exists {
		depnode.FlagStates = globalflags;
	}
	
	//Find current rdeps and add them to the graph
	
	dep_info := repo.GetInstalledByName(dep, destdir)
	if dep_info != nil {
		rdeps := libspack.UninstallList(dep_info.PkgInfo)
		for _, rdep := range rdeps {
			current_info, repo := libspack.GetPackageInstalledByName(rdep.Control.Name, destdir)
			parentnode := New(current_info.Control, repo)
			
			all_flags := current_info.PkgInfo.ComputedFlagStates()
			all_deps := current_info.Control.ParsedDeps()
			deps := all_deps.EnabledFromFlags(all_flags)
			for _, d := range deps {
				if d.Name == dep {
					depnode.MakeParentProud(parentnode, d, false)
					break;
				}
			}
		}
	}
	
	if !depnode.SatisfiesParents() { 
		log.Error(dep, " unable to satisfy parents") //TODO more info
//		return nil
	}
	
	return depnode
}
func (pdl *PkgDepList) ToInstall(destdir string) *PkgDepList {
	newl := make(PkgDepList, 0)
	
	for _, pkg := range *pdl {
		if !pkg.ForgeOnly && !pkg.Repo.IsInstalled(pkg.PkgInfo(), destdir) {
			newl.Append(pkg)
		}
	}
	
	return &newl
}
func (pdl *PkgDepList) Find(name string) *PkgDep {
	for _, pkg := range *pdl {
		if pkg.Control.Name == name {
			return pkg
		}
	}
	return nil
}