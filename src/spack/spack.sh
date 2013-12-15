#!/bin/bash

source /usr/lib/spack/libspack
set -e
basedir="/"
defaults=false

function indirect() {
	eval echo "\$${1}"
}

function set_ind() {
	local var="$1"
	shift
	local stuff="$@"
	eval $var="\"$stuff\""
}

function mark_bin() {
	set_ind ${1}_bin $2
}
function bin_marked() {
	! str_empty $(indirect ${1}_bin) && $(indirect ${1}_bin)
}
function mark_src() {
	set_ind ${1}_src $2
}
function src_marked() {
	! str_empty $(indirect ${1}_src) && $(indirect ${1}_src)
}

bdep_sp=4
function bdep_s() {
	for i in $(eval echo {0..$bdep_sp}); do
		echo -n '-'
	done
}

function dep_check() {
	local name="$1"
	local base="$2"
	local pie="$(get_pie $name)"
	
	if str_empty $pie; then
		echo -n "$name "
		return
	fi
	
	local forge_deps="$(pie_info $pie bdeps)"
	local wield_deps="$(pie_info $pie deps)"
	local dep
	
	log DEBUG $(bdep_s) "TRY $name"
	
	bdep_sp=$(($bdep_sp + 4))
	trap "bdep_sp=$(($bdep_sp - 4)); log DEBUG $(bdep_s) END $name" RETURN
	
	#If we have already been marked as bin, we are done here
	if bin_marked $name; then
		log DEBUG $(bdep_s) "Exists bin $name"
		return
	fi
	
	#If we are a src package, that has not been marked bin, we need a binary version of ourselves to compile ourselves.
	#We are in our own bdeb tree, should only happen for $base if we are having a good day
	if src_marked $name; then
		log DEBUG $(bdep_s) "Exists src $name"
		log DEBUG $(bdep_s) "Mark bin $name"
		mark_bin $name true
		for dep in $wield_deps; do
			dep_check $dep $base
		done
		if ! file_exists $(get_spakg $name); then
			log ERROR "Must have a binary version of $name to build this package"
			echo -n "$name "
		fi
		return
	fi
	
	# We are a package that has a binary version
	if file_exists $(get_spakg $name) && [ "$name" != "$base" ]; then
		log DEBUG $(bdep_s) "Binary $name"
		mark_bin $name true
		if ! $no_bdeps; then
			for dep in $wield_deps; do
				dep_check $dep $base
			done
		fi
		return
	#We are a package that only available via src
	else
		log DEBUG $(bdep_s) "Source $name"
		log DEBUG $(bdep_s) "$name has $bdeps"
		#there is only a src version of us available
		mark_src $name true

		if ! $no_bdeps; then
			for dep in $forge_deps; do
				dep_check $dep $base
			done
		fi
		if [ "$name" != "$base" ]; then
			for dep in $wield_deps; do
				dep_check $dep $base
			done
		fi
		mark_src $name false
		log DEBUG $(bdep_s) "UNSource $name"
		#After this part of the tree we will have a bin version
		mark_bin $name true
		return
	fi
}

function get_pie() {
	local pkg="$1"
	for repo in $repos_dir/*; do
		local file="$repo/$pkg.pie"
		if file_exists $file; then
			echo $file
			return 0
		fi
	done
	log ERROR "Unable to find a pie file for $pkg"
	return 1
}

function get_spakg() {
	local pkg="$1"
	for repo in $spakg_cache_dir/*; do
		local file="$repo/$pkg-*.spakg"
		if file_exists $file; then
			echo $file
			return
		fi
	done
}

#Usage: is_package_installed package_name
function is_package_installed() {
	[ -d $basedir/$spakg_installed_dir/$1/ ]
}

#Usage: set_package_installed spakg
function set_package_installed() {
	local spakg="$1"
	local name=$(spakg_info $spakg name)
	
	mkdir -p $basedir/$spakg_installed_dir/$name
	spakg_part $spakg manifest.txt > $basedir/$spakg_installed_dir/$name/manifest.txt
	spakg_part $spakg pkginfo > $basedir/$spakg_installed_dir/$name/pkginfo
}

#Usage: package_file name file
function package_file() {
	local name="$1"
	local file="$2"
	echo $basedir/$spakg_installed_dir/$name/$file
}

#Usage: set_package_removed package_name
function set_package_removed() {
	rm -rf $basedir/$spakg_installed_dir/$1/
}

function spack_wield_forge_options() {
	local newopts=""
	local option next
	while option="$1"; next="$2"; shift; ! str_empty $option; do
		case $option in
			-d|--basedir)
				basedir="$next"
				shift
			;;
			-r|--reinstall)
				wield_reinstall=true
			;;
			--nobdeps)
				no_bdeps=true
			;;
			--defaults|-y)
				defaults=true
			;;
			*)
				newopts="$newopts $option"
			;;
		esac
	done
	echo $newopts
}

wield_no_check_deps=false
no_bdeps=false
wield_reinstall=false
function spack_wield() {
	require_root
	local skip_deps=false
	local package
	local file
	
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
		;;
	esac
	
	set -- $(spack_wield_forge_options $@)
	
	if str_empty $file; then
		if is_package_installed $package  && ! $wield_reinstall; then
			log INFO "$package already installed, skipping"
			return
		fi
		file=$(get_spakg $package)
		if ! file_exists "$file"; then
			echo "$package is not available in binary form."
			if $defaults || $(ask_yesno true "Do you wish to forge the package?"); then
				echo "OK, building package"
				spack_forge $package $@
				file=$(get_spakg $package)
			else
				log ERROR "Unable to continue, exiting."
				exit 1
			fi
		fi
	fi
	
	if ! file_exists "$file"; then
		log ERROR "$file has not been forged"
		exit 1
	fi
	
	local name=$(spakg_info $file name)
	local deps_checked=""
	
	if is_package_installed $package  && ! $wield_reinstall; then
		log INFO "$package already installed, skipping"
		return
	fi
	
	if ! $wield_no_check_deps; then
		deps_checked=$(dep_check $name)
	fi
	
	if str_empty $deps_checked; then
		if ! $wield_reinstall; then
			local dep
			for dep in $(spakg_info $file deps); do
				log DEBUG $name installing $dep
				spack_wield $dep $@
			done
		else
			log INFO "Skipping installing deps for $package"
		fi
	else
		log ERROR "Unresolved Dependencies: $deps_checked!"
		exit 1
	fi
	wield $file $@ --basedir $basedir
	set_package_installed $file
}

function spack_forge() {
	local output=""
	
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
			require_root
			package="$1"
			shift
			file=$(get_pie $package)
			#hack repo for now
			#TODO: figure out what repo we are grabbing the package from
			#	maybe like 2 funcs, get_pkg_repo $pkg && get_pkg $pkg $repo
			mkdir -p $spakg_cache_dir/Core/
			output="$spakg_cache_dir/Core/$package-$(pie_info $file version).spakg"
			if ! file_exists $file; then
				log ERROR "Unable to find $package in a repository."
				exit 1
			fi
		;;
	esac
	

	set -- $(spack_wield_forge_options $@)
	
	if ! $no_bdeps; then
		local name=$(pie_info $file name)
		local unresolved=""
		if ! $wield_no_check_deps; then
			unresolved=$(dep_check $name $name)
		fi
		if str_empty $unresolved; then
			local old_basedir="$basedir"
			basedir="/" #Install bdeps in the host system
			for dep in $(pie_info $file bdeps); do
				spack_wield $dep $@
			done
			basedir="$basedir"
		else
			log ERROR "Unresolved Dependencies!"
			exit 1
		fi
	fi
	forge $file $@ $output
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
	wield, install    [-f,--file] [--no-check]
	purge, remove
	clear
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
	if str_empty "$option"; then
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
			spack_forge $@
			exit 0
			;;
		wield|install)
			require_root
			spack_wield $@
			exit 0
			;;
		purge|remove)
			require_root
			local name="$1"
			#copy pasta!
			local option next
			while option="$1"; next="$2"; shift; ! str_empty $option; do
				case $option in
					-d|--basedir)
						basedir="$next"
						shift
					;;
				esac
			done
			if is_package_installed $name; then
				log WARN "Purging $name"
				
				for i in $(cat $(package_file $name manifest.txt) | awk '{ print $2 }'); do
					log_cmd DEBUG rm $basedir/$i -rvf
					log INFO "Removing $basedir/$i"
				done
				set_package_removed $name
			else
				log ERROR "$name is not installed"
			fi
			;;
		clean)
			require_root
			package=$1
			file="$(ls $spakg_cache_dir/*/$package*.spakg)"
			log DEBUG "Purging $file"
			rm $file
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
