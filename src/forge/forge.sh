#!/bin/bash

source /usr/lib/spack/libspack

#Globals
pretend=false
run_test=false

MAKEFLAGS="-j4"

tmp_dir="/tmp/forge/$$"
src_dir="$tmp_dir/src"
dest_dir="$tmp_dir/fs"

function none() { return 0; }

#Default
default_func="log ERROR Invalid Function; exit -1"
function default() {
	failexit $default_func
}
function set_default() {
	default_func="$1"
}


function unpack() {
	local archive="$1"
	local cmd="log ERROR Unable to extract $archive; exit 1;"
	local flags=""
	case $archive in
		*.tar*)
			log DEBUG "Extracting $archive using tar"
			cmd="tar"
			if $log_debug; then
				flags="-xvf"
			else
				flags="-xf"
			fi
			;;
		*.zip)
			log INFO "Unzipping $archive"
			
			if ! $log_debug; then
			    flags="-qq"
			fi
			;;
	esac
	failexit $cmd $flags $archive
}

# Usage: fetch_func $src
function fetch_func() {
	case $src in
		*.git)
			log DEBUG "Using git to clone $src"
			local flags=""
			if ! $log_debug; then
				flags="-q"
			fi
			git clone $src $flags
			cd $srcdir
			;;
		http://*|ftp://*)
			log DEBUG "Downloading $src with wget"
			local flags=""
			if ! $log_debug; then
				flags="-q"
			fi
			wget $src $flags
			unpack $(ls)
			cd $srcdir
			;;
		*)
			log ERROR "Unknow format!"
			;;
	esac
}
function fetch() { default; }

function configure_func() {
	./configure
}
function configure() { default; }


function build_func() {
	make $MAKEFLAGS
}
function build() { default; }


function testpkg_func() {
	make test
}
function testpkg() { default; }


function installpkg_func() {
	make DESTDIR=$dest_dir install
}
function installpkg() { default; }


function run_part() {
	local part="$1"
	set_default "${part}_func" 
	breaker
	log INFO "Running $part"
	log_cmd INFO failexit $part
}

function create_pkginstall() {
	cat >> $tmp_dir/pkginstall.sh <<EOT
$(declare -f pre_install)

$(declare -f post_install)
EOT
}

function create_package() {
	breaker
	local fs_rel="fs.tar"
	local fs="$tmp_dir/$fs_rel"
	
	local manifest_rel="manifest.txt"
	local manifest="$tmp_dir/$manifest_rel"
	
	local pkg_install_rel="pkginstall.sh"

	local result="$PWD/$name-$version.spakg"
	
	log INFO "Creating Package"
	cd $dest_dir
		tar -cf $fs *
		find . -type f | xargs md5sum > "${manifest}"
	cd - > /dev/null
	
	cd $tmp_dir
		tar -cf $result $fs_rel $manifest_rel $pkg_install_rel
	cd - > /dev/null
}

function setup() {
	mkdir -p $src_dir
	mkdir -p $dest_dir
}

function cleanup() {
	rm -rf $tmp_dir
}

function create_pkginfo() {
	cat > ./$name.pkginfo <<EOT
name="$name"
version="$version"
info="$desc"
homepage="$url"
flags="$flags"
deps="$deps"
message=""

hooks=""
EOT
}

# Usage: forge file.pie
function forge() {
	local package=$1
	
	echo $(color GREEN "Forging package $package in the heart of a star.")
	log WARN "This can be a dangerous operation, please read the instruction manual to prevent a black hole."
	$pretend && log INFO "(but not really, since pretend is set)"
	
	setup
	
	#todo scope vars with wrapper
	source $package
	
	# TODO check for all vars present and correct
	
	
	local wd="$PWD"
	cd $src_dir
	
	run_part fetch
	run_part configure
	run_part build
	if $run_test; then
		run_part testpkg
	fi
	run_part installpkg
	cd $wd
	
	create_pkginstall
	create_pkginfo
	create_package
	
	cleanup
	
	breaker
	echo $(color GREEN "$name forged successfully")
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
	
	if [ -z "$package" ]; then
		log ERROR "You must specify a package!"
		usage
	else
		forge $package
	fi
}

main $@
