/*
Src
	provides Templates
	generated Controls from teplates
	
	CACHE
	spakgs
	
Avail for build := Controls
Avail for install := Controls
Install no build := spakgs in cache


Src + Bin
	provides Templates
	provides PkgSet
	generated Controls from templates
	
	SEPARATE 
	provides spakgs
	
	CACHE
	spakgs
	
Avail for build := Controls
Avail for install := Controls
Install no build := spakgs in cache and PkgSets

Bin
	provides PkgSet
	
	SEPARATE
	provides spakgs
	
Avail for build := none
Avail for install := Controls
Install no build := PkgSets

*/

package repo

import (
	"libspack/pkginfo"
	"libspack/control"
	"libspack/helpers/json"
)

//Sorted by pkgversion
type ControlMap map[string] control.ControlList

// Map<name, map<version>>
type TemplateFileMap map[string] map[string] string

// Map<name-version, List<PkgInfo>>
type PkgInfoMap map[string][]pkginfo.PkgInfo

// Map<name-version, Tuple<control,pkginfo,hashlist>>
type PkgInstallSetMap map[string]PkgInstallSet

type Repo struct {
	Name string
	Description string
	//Buildable
	RemoteTemplates string	//Templates
	//Installable (pkgset + spakg)
	RemotePackages string	//Control + PkgInfo
	Version string
	
	//Private NOT SERIALIZED
	controls      *ControlMap
	templateFiles *TemplateFileMap
	fetchable     *PkgInfoMap
	installed     *PkgInstallSetMap
}

/*
Serialization
*/
func (repo *Repo) ToFile(filename string) error {
	return json.EncodeFile(filename, true, repo)
}

func FromFile(filename string) (*Repo, error) {
	var repo Repo
	err := json.DecodeFile(filename, &repo)
	
	if err == nil {
		repo.LoadCaches() 
	}
	
	return &repo, err
}
