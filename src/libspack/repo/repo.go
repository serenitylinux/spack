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
	"fmt"
	"regexp"
	"os"
	"io/ioutil"
	"libspack/httphelper"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/hash"
	"libspack/log"
	"libspack/gitrepo"
)

import . "libspack/misc"
import json "libspack/jsonhelper"



type PkgSet struct {
	Control control.Control
	PkgInfo pkginfo.PkgInfo
}
func (p *PkgSet) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}
func PkgSetFromFile(filename string) (*PkgSet, error) {
	var i PkgSet
	err := json.DecodeFile(filename, &i)
	return &i, err
}

type PkgInstallSet struct {
	Control control.Control
	PkgInfo pkginfo.PkgInfo
	Hashes  hash.HashList
}
func (p *PkgInstallSet) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}
func PkgInstallSetFromFile(filename string) (*PkgInstallSet, error) {
	var i PkgInstallSet
	err := json.DecodeFile(filename, &i)
	return &i, err
}


//Sorted by pkgversion
type ControlMap map[string] control.ControlList

// Map<name, map<version>>
type TemplateFileMap map[string] map[string] string

// Map<name-version, List<PkgInfo>>
type PkgInfoMap map[string][]pkginfo.PkgInfo

// Map<name-version, Tuple<control,pkginfo,hashlist>>
type PkgInstallSetMap map[string]PkgInstallSet

const (
	TemplatesDir  = "/var/lib/spack/templates/"	//Downloaded templates
	PackagesDir   = "/var/lib/spack/packages/"	//Downloaded controls and pkginfos
	InstallDir	  = "/var/lib/spack/installed/" //Installed (Pkginfo + controll)s
	ReposCacheDir = "/var/cache/spack/repos/"	//Generated control lists from Templates and Packages
	SpakgDir      = "/var/cache/spack/spakg/"	//Downloaded/build spakgs
)

type Repo struct {
	Name string
	Description string
	//Buildable
	RemoteTemplates string	//Templates
	//Installable (pkgset + spakg)
	RemotePackages string	//Control + PkgInfo
	Version string
	
	//Private NOT SERIALIZED
	controls      ControlMap
	templateFiles TemplateFileMap
	fetchable     PkgInfoMap
	installed     PkgInstallSetMap
}

func (repo *Repo) ToFile(filename string) error {
	return json.EncodeFile(filename, true, repo)
}

func FromFile(filename string) (Repo, error) {
	var repo Repo
	err := json.DecodeFile(filename, &repo)
	
	if err == nil {
		repo.LoadCaches() 
	}
	
	return repo, err
}


func (repo *Repo) templatesDir() string {
	return TemplatesDir + repo.Name + "/"
}
func (repo *Repo) packagesDir() string {
	return PackagesDir + repo.Name + "/"
}
func (repo *Repo) controlCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-Controls.json"
}
func (repo *Repo) pkgInfoCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-PkgInfo.json"
}
func (repo *Repo) templateListCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-Templates.json"
}
func (repo *Repo) installedPkgsDir() string {
	return InstallDir + repo.Name + "/"
}

func cloneRepo(remote string, dir string, name string) {
	switch {
		case GitRegex.MatchString(remote):
			os.MkdirAll(dir, 0755)
			err := gitrepo.CloneOrUpdate(remote, dir)
			if err != nil {
				log.WarnFormat("Update repository %s %s failed: %s", name, remote, err)
			}
		case HttpRegex.MatchString(remote):
			os.MkdirAll(dir, 0755)
			listFile := "packages.list"
			err := httphelper.HttpFetchFileProgress(remote + listFile, dir + listFile, false)
			if err != nil {
				log.Warn(err, remote + listFile)
				return
			}
			
			list := make([]string, 0)
			err = json.DecodeFile(dir + listFile, &list)
			if err != nil {
				log.Warn(err)
				return
			}
			
			for _, item := range list {
				src := remote + "/info/" + item
				err = httphelper.HttpFetchFileProgress(src, dir + item, false)
				if err != nil {
					log.Warn("Unable to fetch %s: %s", err)
				}
			}
		case RsyncRegex.MatchString(remote):
			log.Warn("TODO rsync repo")
		default:
			log.WarnFormat("Unknown repository format %s: '%s'", name, remote)
	}
}

func (repo *Repo) RefreshRemote() {
	if repo.RemoteTemplates != "" {
		cloneRepo(repo.RemoteTemplates, repo.templatesDir(), repo.Name)
	}
	if repo.RemotePackages != "" {
		cloneRepo(repo.RemotePackages, repo.packagesDir(), repo.Name)
	}
	
	repo.UpdateCaches()
}

func (repo *Repo) UpdateCaches() {
	//if we have remote templates
	if repo.RemoteTemplates != "" {
		repo.updateControlsFromTemplates()
	// else if we just have remote controls and prebuilt packages
	} else if repo.RemotePackages != "" {
		repo.updateControlsFromRemote()
	}
	
	if repo.RemotePackages != "" {
		repo.updatePkgInfosFromRemote()
	}
}

//Will also populate template list
func (repo *Repo) updateControlsFromTemplates() {
	//Generates new list and writes to cache
	repo.controls = make(ControlMap)
	
	dir := repo.templatesDir()
	
	templates, err := ioutil.ReadDir(dir)
	if err != nil {
		log.WarnFormat("Unable to load repo %s's templates: %s", repo.Name, err)
		return
	}
	
	for _, templateFile := range templates {
		tfAbs := dir + templateFile.Name()
		c, err := control.GenerateControlFromTemplateFile(tfAbs)
		
		if err != nil {
			log.WarnFormat("Invalid template in repo %s : %s", repo.Name, err)
			continue
		}
		
		// Initialize list of controls for current name if nessesary
		if _, exists := repo.controls[c.Name]; !exists {
			repo.controls[c.Name] = make(control.ControlList, 0)
		}
		
		if _, exists := repo.templateFiles[c.Name]; !exists {
			repo.templateFiles[c.Name] = make(map[string]string)
		}
		
		repo.templateFiles[c.Name][c.Version] = tfAbs
		repo.controls[c.Name] = append(repo.controls[c.Name], *c)
	}
	
	json.EncodeFile(repo.controlCacheFile(), true, repo.controls)
	json.EncodeFile(repo.templateListCacheFile(), true, repo.templateFiles)
}

func (repo *Repo) updateControlsFromRemote() {
	// finds all files in remote dir and writes to cache
	repo.controls = make(ControlMap)
	
	dir := repo.packagesDir()
	
	controls, err := ioutil.ReadDir(dir)
	if err != nil {
		log.WarnFormat("Unable to load repo %s's controls: %s", repo.Name, err)
		return
	}
	
	controlRegex := regexp.MustCompile(".*.pkgset")
	
	for _, cFile := range controls {
		if !controlRegex.MatchString(cFile.Name()) {
			continue
		}
		
		ps, err := PkgSetFromFile(dir + cFile.Name())
		
		if err != nil {
			log.WarnFormat("Invalid control %s in repo %s", cFile.Name(), repo.Name)
			continue
		}
		c := ps.Control
		
		if _, exists := repo.controls[c.Name]; !exists {
			repo.controls[c.Name] = make(control.ControlList, 0)
		}
		repo.controls[c.Name] = append(repo.controls[c.Name], c)
	}
	
	json.EncodeFile(repo.controlCacheFile(), true, repo.controls)
}

func (repo *Repo) updatePkgInfosFromRemote() {
	//Generates new list and writes to cache
	repo.fetchable = make(PkgInfoMap)
	
	dir := repo.packagesDir()
	
	pkginfos, err := ioutil.ReadDir(dir)
	if err != nil {
		log.WarnFormat("Unable to load repo %s's pkginfos: %s", repo.Name, err)
		return
	}
	
	pkginfoRegex := regexp.MustCompile(".*.pkgset")
	
	for _, pkiFile := range pkginfos {
		if !pkginfoRegex.MatchString(pkiFile.Name()) {
			continue
		}
		ps, err := PkgSetFromFile(dir + pkiFile.Name())
		
		if err != nil {
			log.WarnFormat("Invalid pkginfo %s in repo %s", pkiFile.Name(), repo.Name)
			continue
		}
		pki := ps.PkgInfo
		
		key := pki.Name + "-" + pki.Version
		if _, exists := repo.fetchable[key]; !exists {
			repo.fetchable[key] = make([]pkginfo.PkgInfo, 0)
		}
		repo.fetchable[key] = append(repo.fetchable[key], pki)
	}
	
	json.EncodeFile(repo.pkgInfoCacheFile(), true, repo.fetchable)
}


func (repo *Repo) LoadCaches() {
	repo.loadControlCache()
	repo.loadPkgInfoCache()
	repo.loadTemplateListCache()
	repo.loadInstalledPackagesList()
}

func (repo *Repo) loadControlCache() {
	log.DebugFormat("Loading controls for %s", repo.Name)
	repo.controls = make(ControlMap)
	cf := repo.controlCacheFile()
	if PathExists(cf) {
		err := json.DecodeFile(cf, &repo.controls)
		if err != nil {
			log.WarnFormat("Could not load control cache for repo %s: %s", repo.Name, err)
		}
	}
}

func (repo *Repo) loadPkgInfoCache() {
	log.DebugFormat("Loading pkginfos for %s", repo.Name)
	repo.fetchable = make(PkgInfoMap)
	pif := repo.pkgInfoCacheFile()
	if PathExists(pif) {
		err := json.DecodeFile(pif, &repo.fetchable)
		if err != nil {
			log.WarnFormat("Could not load pkginfo cache for repo %s: %s", repo.Name, err)
		}
	}
}

func (repo *Repo) loadTemplateListCache() {
	log.DebugFormat("Loading templates for %s", repo.Name)
	repo.templateFiles = make(TemplateFileMap)
	tlf := repo.templateListCacheFile()
	if PathExists(tlf) {
		err := json.DecodeFile(tlf, &repo.templateFiles)
		if err != nil {
			log.WarnFormat("Could not load template list cache for repo %s: %s", repo.Name, err)
		}
	}
}

func (repo *Repo) loadInstalledPackagesList() {
	log.DebugFormat("Loading installed packages for %s", repo.Name)
	
	repo.installed = make(PkgInstallSetMap)
	
	dir := repo.installedPkgsDir()
	
	if !PathExists(dir) {
		os.MkdirAll(dir, 0755)
	}
	
	filelist, err := ioutil.ReadDir(dir)
	if err != nil {
		log.ErrorFormat("Unable to load repo %s's installed packages: %s", repo.Name, err)
		log.Warn("This is a REALLY bad thing!")
		return
	}
	
	pkgsetRegex := regexp.MustCompile(".*.pkgset")
	
	for _, file := range filelist {
		if !pkgsetRegex.MatchString(file.Name()) {
			continue
		}
		
		ps, err := PkgInstallSetFromFile(dir + file.Name())
		
		if err != nil {
			log.ErrorFormat("Invalid pkgset %s in repo %s", file.Name(), repo.Name)
			log.Warn("This is a REALLY bad thing!")
			continue
		}
		
		repo.installed[ps.Control.Name] = *ps
	}
}

func (repo *Repo) GetAllControls() ControlMap {
	return repo.controls
}

func (repo *Repo) GetControls(pkgname string) (control.ControlList, bool) {
	res, e := repo.GetAllControls()[pkgname]
	return res, e
}

func (repo *Repo) GetLatestControl(pkgname string) (*control.Control, bool) {
	c, exists := repo.GetControls(pkgname)
	var res *control.Control = nil
	
	if exists {
		res = &c[0]
	}
	return res, exists
}

func (repo *Repo) GetAllTemplates() TemplateFileMap {
	return repo.templateFiles
}

func (repo *Repo) GetTemplateByControl(c *control.Control) (string, bool) {
	byName, exists := repo.templateFiles[c.Name]
	if !exists { return "", false }
	byVersion := byName[c.Version]
	if !exists { return "", false }
	return byVersion, true
}

func (repo *Repo) GetSpakgOutput(c *control.Control) string {
	if !PathExists(SpakgDir + repo.Name) {
		os.MkdirAll(SpakgDir + repo.Name, 0755)
	}
	return SpakgDir + fmt.Sprintf("%s/%s-%s.spakg", repo.Name, c.Name, c.Version)
}

func (repo *Repo) HasSpakg(c *control.Control) bool {
	return PathExists(repo.GetSpakgOutput(c))
}

func (repo *Repo) HasTemplate(c *control.Control) bool {
	_, exists := repo.GetTemplateByControl(c)
	return exists
}

func (repo *Repo) Install(c control.Control, p pkginfo.PkgInfo, hl hash.HashList, basedir string) error {
	ps := PkgInstallSet { c, p, hl }
	err := os.MkdirAll(basedir + repo.installedPkgsDir(), 0755)
	if err != nil {
		return err
	}
	err = ps.ToFile(basedir + fmt.Sprintf("%s/%s-%s.pkgset", repo.installedPkgsDir(), c.Name, c.Version))
	repo.loadInstalledPackagesList()
	return err
}

func (repo *Repo) IsInstalled(c *control.Control, basedir string) bool {
	return PathExists(basedir + fmt.Sprintf("%s/%s-%s.pkgset", repo.installedPkgsDir(), c.Name, c.Version))
}

func (repo *Repo) RDeps(c *control.Control) []PkgInstallSet {
	pkgs := make([]PkgInstallSet,0)
	
	var inner func (*control.Control)
	
	inner = func (cur *control.Control) {
		fmt.Println(cur)
		for _, pkg := range pkgs {
			if pkg.Control.Name == cur.Name {
				return
			}
		}
		
		for _, set := range repo.installed {
			for _, dep := range set.Control.Deps {
				if dep == cur.Name {
					pkgs = append(pkgs, set)
					inner(&set.Control)
				}
			}
		}
	}
	
	inner(c)
	
	return pkgs
}

func (repo *Repo) Uninstall(c *control.Control) error{
	list := repo.RDeps(c)
	
	if len(list) == 0 {
		fmt.Println("No deps")
		return nil
	}
	
	for _, set := range list {
		fmt.Println("Remove: ", set.Control.Name)
	}
	return nil
}
