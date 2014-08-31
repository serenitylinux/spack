package repo

import "os"

const (
	TemplatesDir  = "/var/lib/spack/templates/"	//Downloaded templates
	PackagesDir   = "/var/lib/spack/packages/"	//Downloaded controls and pkginfos
	InstallDir	  = "/var/lib/spack/installed/" //Installed (Pkginfo + controll)s
	ReposCacheDir = "/var/cache/spack/repos/"	//Generated control lists from Templates and Packages
	SpakgDir      = "/var/cache/spack/spakg/"	//Downloaded/build spakgs
)

/*
Package Directories
*/
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
func (repo *Repo) spakgDir() string {
	return SpakgDir + repo.Name + "/"
}