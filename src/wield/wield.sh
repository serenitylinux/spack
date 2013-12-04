#!/bin/bash

source /usr/lib/spack/libspack

pretend=false

tmp_dir="/tmp/$$"
manifest="$tmp_dir/manifest.txt"
fs_dir="$tmp_dir/fs"


function setup() {
	log DEBUG "Working directory: $tmp_dir"
	
	mkdir $tmp_dir
	mkdir $fs_dir
}

function cleanup() {
	log DEBUG "Removing directory: $tmp_dir"
	log_cmd DEBUG rm -v $tmp_dir -rf
}

#Usage: extract pkg
function extract() {
	local package="$1"
	
	log INFO "Extracting Package:"
	log_cmd DEBUG tar -xvf $package -C $tmp_dir
	
	if ! file_exists $manifest; then
		log ERROR "Invalid package, missing manifest!"
		exit 1;
	fi
	
	if file_exists $tmp_dir/fs.tar; then
		log INFO
		log INFO "Extracting FS:"
		log_cmd DEBUG tar -xvf $tmp_dir/fs.tar -C $fs_dir
	else
		log ERROR "Invalid package, missing fs!"
		exit 1;
	fi
	
	
	local pkginfo="$(ls $tmp_dir/pkginfo)"
	if file_exists $pkginfo; then
		source $pkginfo
	else
		log ERROR "Invalid package, could not find pkginfo!"
		exit -1
	fi
}

function check() {
	log INFO "Checking integrity and collisions"
	local msum
	
	cd $fs_dir
	
	log DEBUG md5sum $manifest
	log_cmd DEBUG md5sum -c $manifest
	
	cd - > /dev/null
	
	#TODO check fs collisions
}

function install_files() {
    if $pretend; then
		return 0
	fi
	
	log INFO "Installing package"

	cd $fs_dir
	for file in $(cd $fs_dir && find .); do
		file=${file#"."}
		if file_exists $fs_dir/$file; then
			log INFO "Installing file: $file"
			if file_exists $file; then
				log INFO "Replacing $file"
				log_cmd DEBUG rm -v $file
			fi
			mkdir -p $(dirname $file)/
			log_cmd DEBUG install -cvD $fs_dir/$file $(dirname $file)/
		fi
	done
	cd - > /dev/null
}

function run_step() {
	local part="$@"
	
	log INFO "Running $part"
	log_cmd INFO breaker
	
	failexit "Section $part failed for package $name, exiting" print_result log_cmd INFO $part
	log INFO
}

#Usage: wield pkg.spkg
function wield_pkg() {
	local package="$1"
	color GREEN "Wielding $package with the force of a $(color RED GOD)!"; echo; echo
	run_step setup
	
	run_step extract $package
	source $tmp_dir/pkginstall.sh
	
	run_step check
	
	if ! $pretend; then
		if func_exists pre_install; then
			run_step pre_install
		fi

		run_step install_files

		if func_exists post_install; then
			run_step post_install
		fi
	fi
	
	run_step cleanup
	
	color GREEN "Your heart is pure and accepts the gift of $name."; echo
}

function usage() {
	cat <<EOT

Usage: $0 [OPTIONS] package.spakg
	-p, --pretend
	-v, --verbose
	-q, --quiet
	-h, --help

EOT
exit 0
}

function main() {
	local option
	local package
	for option in $@; do
		case $option in
			-p|--pretend)
				pretend=true;;
			-q|--quiet)
				set_log_levels ERROR
				;;
			-v|--verbose)
				set_log_levels DEBUG
				;;
			-h|--help)
				usage;;
			*.spakg)
				if file_exists $option; then
					package=$option
				else
					log ERROR "Unknown package: $option"
					exit 1
				fi;;
			*)
				log ERROR "Unrecognized option: $option"
				usage;;
		esac
	done
	
	if ! $pretend; then
		require_root
	fi
	
	if str_empty "$package"; then
		log ERROR "You must specify a package!"
		usage
	fi
	wield_pkg $package
}

main $@