package depres

//Containing the Insanity to a single file (hopefully)
//Christian Mesh Feb 2014

//TODO version checking
//TODO check valid set of flags on a per package basis

import (
	"strings"
	"libspack/log"
	"libspack/dep"
	"libspack/depres/pkgdep"
)

type DepResParams struct {
	IsForge bool
	IsReinstall bool
	IgnoreBDeps bool
	DestDir string
}

var indent int = 0
func DepTree(node *pkgdep.PkgDep, graph *pkgdep.PkgDepList, params DepResParams) bool {
	indent++
	defer func() { indent-- }()
	
	debug := func (s string) {
		log.DebugFormat("%s %s %s", strings.Repeat("\t", indent), node.Control.UUID(), s)
	}
	debug("check")
	
	//We are already installed
	//And not a reinstall
	//And not being built
	if !params.IsReinstall && !params.IsForge && node.IsInstalled(params.DestDir) {
		debug("already installed")
		return true
	}
	
	//We do not need to be rechecked
	if !node.Dirty {
		debug("clean")
		return true
	}
	
	//We are being built and do not care about bdeps, I think we are done here
	if params.IsForge && params.IgnoreBDeps {
		debug("Ignore bdeps")
		return true
	}
	
	node.Dirty = false //We will be making sure we are clean in the next step
	rethappy := true   //Clap your hands!
	
	var deps []dep.Dep
	if params.IsForge {
		deps = node.Control.ParsedBDeps()
	} else {
		deps = node.Control.ParsedDeps()
	}
	isbdep := params.IsForge //Make a copy of isForge for later
	params.IsForge = false
	params.IsReinstall = false
	
	for _, dep := range deps {
		debug("Require: " + dep.Name)
		
		depnode := graph.Find(dep.Name)
		if depnode == nil {
			depnode = graph.Add(dep.Name, params.DestDir)
		}
		
		if depnode.ForgeOnly {
			debug("too far down the rabbit hole: "+ dep.Name)
			//TODO log.Error(
			rethappy = false
			continue
		}
		
		//Will set to dirty if changed
		if dep.Flags != nil && !depnode.MakeParentProud(node, dep, isbdep) {
			debug("Changed "+ dep.Name)
			rethappy = false
			continue
		}
		
		//Continue down the rabbit hole
		if !DepTree(depnode, graph, params) {
			debug("Not Happy "+ dep.Name)
			rethappy = false
		}
	}
	debug("done")
	return rethappy
}

func FindToBuild(graph *pkgdep.PkgDepList, params DepResParams) (*pkgdep.PkgDepList, bool) {
	log.Debug("Finding packages to build:")
	
	orderedlist := make(pkgdep.PkgDepList, 0)
	visitedlist := make(pkgdep.PkgDepList, 0)
	
	happy := findToBuild(graph, &orderedlist, &visitedlist, params)
	visitedlist.Reverse() //See diagram below
	
	return &visitedlist, happy
}

func findToBuild(graph, orderedtreelist, visitedtreelist *pkgdep.PkgDepList, params DepResParams) bool {
	indent++
	defer func() { indent-- }()
	
	debug := func (s string) {
		log.DebugFormat("%s %s", strings.Repeat("\t", indent), s)
	}
	
	//list of packages to build
	tobuild := make(pkgdep.PkgDepList, 0)
	
	for _, node := range *graph {
		debug("Check " + node.PkgInfo().UUID())
		//TODO Does the next function call need to care about params.DestDir?
		if !node.SpakgExists() && !node.IsInstalled(params.DestDir) || node.ForgeOnly{
			tobuild.Append(node)
		}
	}
	
	happy := true //If you are happy and you know it clap your hands!!
	params.IsForge = true
	for _, node := range tobuild {
		//We have not already been "built"
		if !visitedtreelist.Contains(node) {
			//Create a new graph representing the build deps of node
			newroot := pkgdep.New(node.Control, node.Repo)
			newroot.FlagStates = node.FlagStates //This *should* be a deep copy
			
			newrootgraph := make(pkgdep.PkgDepList, 0)
			newroot.Graph = &newrootgraph
			
			//mark newroot read only
			newroot.ForgeOnly = true
			
			//Add ourselves to existing builds
			visitedtreelist.Append(newroot)
			
			if !DepTree(newroot, newroot.Graph, params) {
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