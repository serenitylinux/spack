package depres

//Containing the Insanity to a single file (hopefully)
//Christian Mesh Feb 2014

//TODO version checking
//TODO check valid set of flags on a per package basis

import (
	"libspack"
	"libspack/log"
	"libspack/dep"
	//"libspack/flag"
	"libspack/flagconfig"
	"libspack/depres/pkgdep"
)

type DepResParams struct {
	IsForge bool
	IsReinstall bool
	IgnoreBDeps bool
	DestDir string
}

func DepTree(node *pkgdep.PkgDep, tree *pkgdep.PkgDep, params DepResParams) bool {
	//We are already installed
	//And not a reinstall
	//And not being built
	if !params.IsReinstall && !params.IsForge && node.IsInstalled(params.DestDir) {
		return true
	}
	
	//We do not need to be rechecked
	if !node.Dirty {
		return true
	}
	
	//We are being built and do not care about bdeps, I think we are done here
	if params.IsForge && params.IgnoreBDeps {
		return true
	}
	
	node.Dirty = false //We will be making sure we are clean in the next step
	rethappy := true
	
	var deps []dep.Dep
	if params.IsForge {
		deps = node.Control.ParsedBDeps()
	} else {
		deps = node.Control.ParsedDeps()
	}
	
	params.IsForge = false
	params.IsReinstall = false
	
	for _, dep := range deps {
		depnode := tree.Find(dep.Name)
		if depnode == nil {
			//Create new pkgdep node
			ctrl, repo := libspack.GetPackageLatest(dep.Name)
			if ctrl == nil {
				log.Error(node.String(), "Unable to find package", dep)
				rethappy = false
				continue
			}
			
			depnode = pkgdep.New(ctrl, repo)
			tree.AllNodes.Append(depnode)
			
			//Add global flags to new depnode
			globalflags, exists := flagconfig.GetAll(params.DestDir)[ctrl.Name]
			if exists && !depnode.MakeParentProud(nil, globalflags) { 
				rethappy = false
				continue
			}
		}
		
		if depnode.ForgeOnly {
			//TODO log.Error(
			rethappy = false
			continue
		}
		
		//Will set to dirty if changed
		if !depnode.MakeParentProud(node, dep.Flags.List) {
			rethappy = false
			continue
		}
		
		//update references from self to depnode and vice versa
		depnode.Parents.Append(node)
		
		//Continue down the rabbit hole
		if !DepTree(depnode, tree, params) {
			rethappy = false
		}
	}
	return rethappy
}

func FindToBuild(graph *pkgdep.PkgDepList, params DepResParams) (*pkgdep.PkgDepList, bool) {
	orderedlist := make(pkgdep.PkgDepList, 0)
	visitedlist := make(pkgdep.PkgDepList, 0)
	
	happy := findToBuild(graph, &orderedlist, &visitedlist, params)
	visitedlist.Reverse() //See diagram below
	
	return &visitedlist, happy
}

func findToBuild(graph, orderedtreelist, visitedtreelist *pkgdep.PkgDepList, params DepResParams) bool {
	//list of packages to build
	tobuild := make(pkgdep.PkgDepList, 0)
	
	for _, node := range *graph {
		//TODO Does the next function call need to care about params.DestDir?
		if !node.SpakgExists() {
			tobuild.Append(node)
		}
	}
	
	happy := true
	params.IsForge = true
	for _, node := range tobuild {
		//We have not already been "built"
		if !visitedtreelist.Contains(node) {
			//Create a new graph representing the build deps of node
			newroot := pkgdep.New(node.Control, node.Repo)
			newroot.FlagStates = node.FlagStates
			
			newrootgraph := make(pkgdep.PkgDepList, 0)
			newroot.AllNodes = &newrootgraph
			
			//mark newroot read only
			newroot.ForgeOnly = true
			
			//Add ourselves to existing builds
			visitedtreelist.Append(newroot)
			
			if !DepTree(newroot, newroot, params) {
				happy = false
				continue
			}
			
			if !findToBuild(&newrootgraph, orderedtreelist, visitedtreelist, params) {
				happy = false
				continue
			}
			
			//We now have our deps in a correct state so we can add ourselves to the order
			orderedtreelist.Append(newroot)
		} else {
			//We have been visited but are not satisfied yet !!! A -> (C, B), B -> (C, A) Invalid
			/*
				A visit
					C visit
						No deps
					C order
					B visit
						C visit and order    == OK
						A visited and !order == NOT OK
					B order
				A order
						
				existing in order signifies that the package is ok to go
			*/
			if !orderedtreelist.Contains(node) {
				happy = false
			}
		}
	}
	return happy
}