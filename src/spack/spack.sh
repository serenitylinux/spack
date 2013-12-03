#!/bin/bash

source /usr/lib/spack/libspack
set -e

function indirect() {
	local foo
	local arg="$1"
	eval foo="\$${arg}"
	echo $foo
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
		log ERROR "$pkg not found $marked"
		exit 1
	fi

	log DEBUG $(dep_tab) "$pkg set=$marked"
	if ! $marked; then
		mark_pkg $pkg
		log DEBUG $(dep_tab) "mark $pkg"
		local deps=$(indirect ${pkg}_deps)
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
	local file="$1"
	for info in $repos_dir/Core/*.pie; do #Pie or Pkginfo?
		load_package_deps $info
	done

	local name deps bdeps
	#TODO HACK
	cd /tmp
	tar --wildcards -xvf  $file *.pkginfo >&2
	cd - > /dev/null
	source $(ls /tmp/*.pkginfo)
	log DEBUG "Starting with $name"
	local to_install=$(dep $name);
	echo "$to_install"
}

function get_pie() {
	local pkg="$1"
	for repo in $repos_dir/*; do
		local file="$repo/$pkg.pie"
		if [ -f $file ]; then
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
		if [ -f $file ]; then
			echo $file
			return
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
					file=$(get_spakg $1)
					shift 
				;;
			esac
			if ! [ -f "$file" ]; then
				log ERROR "$package has not been built"
				exit 1
			fi
			local dep dfile
			local pkg_deps=$(deps $file)
			log INFO $pkg_deps

			for dep in $pkg_deps; do
				if [ -z "$dfile"] || [ ! -f $dfile ]; then
					spack forge $dep
				fi
			done

			for dep in $pkg_deps; do
				dfile=$(get_spakg $dep)
				if [ ! -z "$dfile"] && [ -f $dfile ]; then
					echo wield $dfile $@
				else
					log ERROR "dep $dep not built"
					exit 1
				fi
			done
			echo wield $file $@
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
