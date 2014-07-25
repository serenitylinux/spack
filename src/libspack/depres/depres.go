package depres

//Containing the Insanity to a single file (hopefully)
//Christian Mesh Feb 2014

//TODO check valid set of flags on a per package basis

import (
	"strings"
	"lumberjack/log"
	"libspack/dep"
	"libspack/depres/pkgdep"
)

type DepResParams struct {
	IsForge bool
	IsReinstall bool
	IgnoreBDeps bool
	DestDir string
}

//TODO log.Error.Println(all the things)
var indent int = 0
func DepTree(node *pkgdep.PkgDep, graph *pkgdep.PkgDepList, params DepResParams) bool {
	indent++
	defer func() { indent-- }()
	
	debug := func (s string) {
		log.Debug.Format("%s %s %s", strings.Repeat("\t", indent), node.Control().UUID(), s)
	}
	debug("check")
	
	//We are already installed exact (checks version and flags as well)
	//And not a reinstall
	//And not being built
	if !params.IsForge && node.IsInstalled(params.DestDir) && !params.IsReinstall {
		debug("already installed")
		return true
	}
	node.IsReinstall = params.IsReinstall;
	
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
	
	var deps dep.DepList
	if params.IsForge {
		deps = node.Control().ParsedBDeps()
	} else {
		deps = node.Control().ParsedDeps()
	}
	
	setflags := node.ComputedFlags()
	deps = deps.EnabledFromFlags(*setflags)
	
//	isbdep := params.IsForge //Make a copy of isForge for later
	params.IsForge = false
	params.IsReinstall = false
	
	//We are new or have been changed
	for _, dep := range deps {
		debug("Require: " + dep.Name)
		
		depnode := graph.Find(dep.Name)
		//We are not part of the graph yet
		if depnode == nil {
			depnode = graph.Add(dep.Name, params.DestDir)
		}
		
		if depnode.ForgeOnly {
			debug("too far down the rabbit hole: "+ dep.Name)
			rethappy = false
			continue
		}
		
		//Will set to dirty if changed and add parent constraint
		if !depnode.AddParent(node, dep) {
			//We can't add this parent constraint
			debug("Cannot change " + dep.Name + " to " + dep.String())
			log.Error.Write([]byte("Conflicting package constraints on " + dep.Name + ":" + "\n"))
			depnode.Constraints.PrintError("\t")
			rethappy = false
			continue
		}
		
		//Continue down the rabbit hole ...
		if !DepTree(depnode, graph, params) {
			debug("Not Happy "+ dep.Name)
			rethappy = false
		}
	}
	debug("done")
	return rethappy
}

func FindToBuild(graph *pkgdep.PkgDepList, params DepResParams) (*pkgdep.PkgDepList, bool) {
	log.Debug.Println("Finding packages to build:")
	
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
		log.Debug.Format("%s %s", strings.Repeat("\t", indent), s)
	}
	
	//list of packages to build
	tobuild := make(pkgdep.PkgDepList, 0)
	
	for _, node := range *graph {
		debug("Check " + node.PkgInfo().PrettyString())
		if !node.SpakgExists() && !node.IsInstalled(params.DestDir) || node.ForgeOnly {
			debug("Build " + node.PkgInfo().PrettyString())
			tobuild.Append(node)
		} else {
			debug("Have " + node.PkgInfo().PrettyString())
		}
	}
	
	happy := true //If you are happy and you know it clap your hands!!
	params.IsForge = true
	for _, node := range tobuild {
		//We have not already been "built"
		if !visitedtreelist.Contains(node.Name) {
			//Create a new graph representing the build deps of node
			newroot := pkgdep.New(node.Name, node.Repo)
			newroot.Constraints = node.Constraints //This *should* be a deep copy
			
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
			if !orderedtreelist.Contains(node.Name) {
				happy = false
			}
		}
	}
	return happy
}