#!/bin/bash

source ../lib.sh

pretend=false

tmp_dir="/tmp/$$"
manifest="$tmp_dir/manifest.txt"
fs_dir="$tmp_dir/fs"


function setup() {
	log INFO "Setup"
	log DEBUG "Working directory: $tmp_dir"
	
	mkdir $tmp_dir
	mkdir $fs_dir
}

function cleanup() {
	local cmd="rm"
	if $log_debug; then
		cmd="$cmd -v"
	fi
	log INFO "Cleaning up"
	log DEBUG "Removing directory: $tmp_dir"
	
	$cmd $tmp_dir -rf
}

#Usage: extract pkg
function extract() {
	log INFO "Extracting $pkg"
	local pkg="$1"
	local cmd
	if $log_debug; then
		cmd="tar -xvf"
	else
		cmd="tar -xf"
	fi
	
	failexit $cmd $pkg -C $tmp_dir
	
	if [ ! -f $manifest ]; then
		log ERROR "Invalid package, missing manifest!"
		exit 1;
	fi
	if [ ! -f $tmp_dir/*.fs.tar ]; then
		log ERROR "Invalid package, missing fs!"
		exit 1;
	fi
	
	failexit $cmd $tmp_dir/*.fs.tar -C $fs_dir
}

function check() {
	log INFO "Checking integrity and collisions"
	local msum
	
	cd $fs_dir
	msum=$(md5sum -c $manifest)
	if [ $? -ne 0 ]; then
		echo $msum | grep -v ": OK"
		log ERROR "Corrupt/missing files!"
		exit 1;
	fi
	
	log DEBUG "md5sum:"
	if $log_debug; then
		echo "$msum"
	fi
	
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
		install -d $file /
	done
	cd - > /dev/null
}

#Usage: wield pkg.spkg
function wield_pkg() {
	local pkg="$1"
	
	setup
	
	extract $pkg
	
	check
	
	if ! $pretend; then
		install_files
	fi
	
	cleanup
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
				log_warn=false
				log_error=true
				log_debug=false
				log_info=false
				;;
			-v|--verbose)
				log_warn=true
				log_error=true
				log_debug=true
				log_info=true
				;;
			-h|--help)
				usage;;
			*.spakg)
				if [ -f $option ]; then
					package=$option
				else
					echo "Unknown package: $option"
					exit 1
				fi;;
			*)
				log ERROR "Unrecognized option: $option"
				usage;;
		esac
	done
	if [ -z "$package" ]; then
		log ERROR "You must specify a package!"
		usage
	fi
	wield_pkg $package
}

main $@