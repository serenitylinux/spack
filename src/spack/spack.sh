#!/bin/bash

source /usr/lib/spack/libspack
set -e

function indirect() {
	eval echo "\$${1}"
}

function set_ind() {
	local var="$1"
	shift
	local stuff="$@"
	eval $var="\"$stuff\""
}

function package_exists() {
	[ ! -z "$(ismarked_pkg $1)" ]
}

gp_list=""
function load_package_deps() {
	local pkg="$1"
	local name deps bdeps
	source $1
	log DEBUG "Loading package $name"
	set_ind ${name}_deps "$deps"
	unmark_pkg $name
	gp_list="$gp_list $name"
}

function get_package_deps() {
	local pkg="$1"
	indirect ${pkg}_deps
}

function mark_pkg() {
	set_ind ${1}_set true
}
function unmark_pkg() {
	log DEBUG "unmarking pkg $1"
	set_ind ${1}_set false
}
function ismarked_pkg() {
	indirect ${1}_set
}

dep_tabs=4
function dep_tab() {
	for i in $(eval echo {0..$dep_tabs}); do
		echo -n '-'
	done
}
function dep() {
	local pkg="$1"
	local marked=$(ismarked_pkg $pkg)
	
	if ! package_exists $pkg; then
		log ERROR "$pkg not found"
		exit 1
	fi

	log DEBUG $(dep_tab) "$pkg set=$marked"
	if ! $marked; then
		local deps=$(indirect ${pkg}_deps)
		mark_pkg $pkg
		log DEBUG $(dep_tab) "Resolving deps for $pkg: $deps"
		
		dep_tabs=$(($dep_tabs + 4))
		for dep in $deps; do
			echo -n "$(dep $dep)"
		done
		dep_tabs=$(($dep_tabs - 4))
		echo "$pkg "
	fi
}

function deps() {
	local name="$1"
	for info in $repos_dir/Core/*.pie; do #Pie or Pkginfo?
		load_package_deps $info
	done
	
	log DEBUG "Starting with $name"
	dep $name
}

function get_pie() {
	local pkg="$1"
	for repo in $repos_dir/*; do
		local file="$repo/$pkg.pie"
		if file_exists $file; then
			echo $file
			return
		fi
	done
	log ERROR "Unable to find a pie file for $pkg"
	exit 1
}

function get_spakg() {
	local pkg="$1"
	for repo in $spakg_cache_dir/*; do
		local file="$repo/$pkg-*.spakg"
		log DEBUG $file
		if file_exists $file; then
			echo $file
			return
		fi
	done
}

function spack_wield() {
	require_root
	
	local file="$1"
	shift
	
	if ! file_exists "$file"; then
		log ERROR "$file has not been forged"
		exit 1
	fi
	
	local pkg_name=$(spakg_info $file name)
	
	local pkg_deps=$(deps $pkg_name)
	log INFO "Dependencies for $pkg_name: $pkg_deps"
	
	local dep
	for dep in $pkg_deps; do
		local dfile=$(get_spakg $dep)
		if ! file_exists $dfile; then
			spack forge $dep $@
		fi
	done
	
	for dep in $pkg_deps; do
		dfile=$(get_spakg $dep)
		if file_exists $dfile; then
			wield $dfile $@
		else
			log ERROR "dep $dep not built"
			exit 1
		fi
	done
}

function refresh_repos() {
	require_root
	for i in /etc/spack/repos/*; do
		local name desc remote version
		source $i

		log INFO "Refreshing repository $name $version"
		log DEBUG "$desc"
		mkdir -p $repos_dir/$name
		cd $repos_dir/$name
		case $remote in 
			*.git)
				log DEBUG "Cloning $remote using git"
				if [ -d .git ]; then
					log_cmd INFO git pull
				else
					log_cmd INFO git clone $remote .
				fi
			;;
			http://|https://|ftp://)
				log DEBUG "Cloning $remote using wget"
				log_cmd INFO wget -r --no-parent $remote
			;;
			rsync://*)
				log DEBUG "Cloning $remote using rsync"
				log_cmd INFO rsync -av $remote .
			;;
			*)
				log ERROR "Invalid repository: $remote"
			;;
		esac
		cd - >/dev/null
	done
}


function usage() {
	cat <<EOT
Usage: $0
	refresh
	upgrade
	forge, build      [-f,--file]
	wield, install    [-f,--file]
	search
	info
	packages
	news
	audit
	-h, --help (this menu)
EOT
}

function spack_options() {
	local option
	for option in $@; do
		case $option in
			-v|--verbose)
				set_log_levels DEBUG
				;;
			-q|--quiet)
				set_log_levels WARN
				;;
		esac
	done
}

function main() {
	local package=""
	local file=""
	local option="$1"
	if [ -z "$option" ]; then
		usage
		exit 1
	fi

	shift

	spack_options $@
	case $option in
		refresh)
			refresh_repos
			exit -1
			;;
		upgrade)
			log ERROR "upgrade option not implemented"
			exit -1
			;;
		-h|--help)
			usage
			exit 0
			;;
		forge|build)
			case $1 in
				-f|--file)
					file="$2"
					shift 2
				;;
				*.pie)
					file="$1"
					shift
				;;
				*)
					package="$1"
					shift
					file=$(get_pie $package)
				;;
			esac
			#hack repo for now
			mkdir -p $spakg_cache_dir/Core
			cd $spakg_cache_dir/Core
			forge $file $@
			cd - > /dev/null
			exit 0
			;;
		wield|install)
			case $1 in
				-f|--file)
					file="$2"
					shift 2
				;;
				*.spakg)
					file="$1"
					shift
				;;
				*)
					package="$1"
					shift
					file=$(get_spakg $package)
					if [ -z "$file" ]; then
						echo "$package is not available in binary form."
						if $(ask_yesno true "Do you wish to forge the package?"); then
							echo "OK, building package"
							spack forge $package $@
							file=$(get_spakg $package)
						else
							log ERROR "Unable to continue, exiting."
							exit 1
						fi
					fi
				;;
			esac
			spack_wield $file $@
			exit 0
			;;
		search)
			log ERROR "Not Implemented"
			exit -1
			;;
		*)
			usage
			exit 1
	esac
}


main $@
