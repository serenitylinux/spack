#!/bin/bash

#Globals
pretend=false
run_test=false

MAKEFLAGS="-j4"

tmp_dir="/tmp/forge/$$"
src_dir="$tmp_dir/src"
dest_dir="$tmp_dir/fs"

#Usage: color COLOR text
function color() {
	local color=$1
	shift
	local text=$@
	local r_color
	
	case $color in
		BLACK|black)
			r_color="0;30";;
		RED|red)
			r_color="0;31";;
		GREEN|green)
			r_color="0;32";;
		BROWN|brown)
			r_color="1;33";;
		BLUE|blue)
			r_color="0;34";;
		PURPLE|purple)
			r_color="0;35";;
		CYAN|cyan)
			r_color="0;36";;
		YELLOW|yellow)
			r_color="1;33";;
		WHITE|white)
			r_color="1;37";;
	esac
	echo -en "\e[${r_color}m${text}\e[0m"
}

log_info=true
log_debug=false
log_warn=true
log_error=true
# TODO move to lib
# Usage: log $level $string
function log() {
	local level="$1"
	shift
	local string="$@"
	case $level in
		INFO)
			$log_info && echo $(color WHITE $@) ;;
		DEBUG)
			$log_debug && color BLUE "debug: " && echo $@ ;;
		WARN)
			$log_warn && color YELLOW "warning: " && echo $@ ;;
		ERROR)
			$log_error && color RED "error: " && echo $@ ;;
	esac
}

#Default
default_func="log ERROR Invalid Function; exit -1"
function default() {
	$default_func
}
function set_default() {
	default_func="$1"
}


function unpack() {
	local archive="$1"
	local cmd="log ERROR Unable to extract $archive; exit 1;"
	case $archive in
		*.tar*)
			log INFO "Extracting $archive using tar"
			if $log_debug; then
				cmd="tar -xvf"
			else
				cmd="tar -xf"
			fi
			;;
		*.zip)
			log INFO "Unzipping $archive"
			cmd="unzip"
			;;
	esac
	echo $cmd
	$cmd $archive
}

# Usage: fetch_func $src
function fetch_func() {
	case $src in
		http://*|ftp://*)
			log INFO "Downloading $src with wget"
			wget $src
			unpack "zlib-${version}.tar.gz"
			cd $(ls | grep -v "tar.gz")
			;;
		*.git)
			log INFO "Using git to clone $src"
			git clone $src
			cd $(ls)
			;;
		*)
			log ERROR "Unknow format!!!"
			;;
	esac
}
function fetch() { default; }

function configure_func() {
	./configure
}
function configure() { default; }


function build_func() {
	fakeroot make $MAKEFLAGS
}
function build() { default; }


function testpkg_func() {
	make test
}
function testpkg() { default; }


function installpkg_func() {
	fakeroot make DESTDIR=$dest_dir install
}
function installpkg() { default; }


function run_part() {
	local part="$1"
	set_default "${part}_func" 
	log DEBUG "Run Part $part"
	$part
}

function create_package() {
	local fs_rel="$name.fs.tar"
	local fs="$tmp_dir/$fs_rel"
	
	local manifest_rel="manifest.txt"
	local manifest="$tmp_dir/$manifest_rel"
	
	local result="$PWD/$name-$version.spakg"
	
	log INFO "Creating Package"
	cd $dest_dir
		tar -cf $fs *
		find . -type f | xargs md5sum > "${manifest}"
	cd -
	
	cd $tmp_dir
		tar -cf $result $fs_rel $manifest_rel
	cd -
}

function setup() {
	mkdir -p $src_dir
	mkdir -p $dest_dir
}

function cleanup() {
	rm -rf $tmp_dir
}

# Usage: forge file.pie
function forge() {
	local package=$1
	
	echo "Forging package $package in the heart of a star."
	log WARN "This can be a dangerous operation, please read the instruction manual to prevent a black hole."
	$pretend && echo "Just kidding :P" #TODO
	
	setup
	
	#todo scope vars with wrapper
	source $package
	
	# TODO check for all vars present and correct
	
	
	local wd="$PWD"
	log DEBUG "cd $src_dir"
	cd $src_dir
	
	run_part fetch
	run_part configure
	run_part build
	if $run_test; then
		run_part testpkg
	fi
	run_part installpkg
	cd $wd
	
	create_package
	
	cleanup
}


function usage() {
	cat <<EOT

Usage: $0 [OPTIONS] package.pie
	-p, --pretend
	-v, --verbose
	-q, --quiet
	-t, --test
	-h, --help

EOT
exit 1
}

function main() {
	local option
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
				echo "Usage!"
				usage;;
			-t|--test)
				run_test=true;;
			*.pie)
				if [ -f $option ]; then
					package=$option
				else
					echo "Unknown package file: $option"
					exit 1
				fi;;
			*)
				log ERROR "Unrecognized option: $option"
				usage;;
		esac
	done
	
	if [ -z "$package"]; then
		log ERROR "You must specify a package!"
		usage
	else
		forge $package
	fi
}





main $@
