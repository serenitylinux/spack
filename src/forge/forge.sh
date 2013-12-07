#!/bin/bash

source /usr/lib/spack/libspack

#Globals
pretend=false
run_test=false

tmp_dir="/tmp/$$/forge"
src_dir="$tmp_dir/src"
dest_dir="$tmp_dir/fs"

STARTDIR="$PWD"

function none() { echo "Nothing to do!"; }

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
	local cmd=""
	case $archive in
		*.tar*)
			log DEBUG "Extracting $archive using tar"
			cmd="tar -xf"
			;;
		*.zip)
			log INFO "Unzipping $archive"
			cmd="unzip"
			;;
		*)
			log ERROR Unable to extract $archive
			return 1
		;;
	esac
	log_cmd DEBUG $cmd $archive
	return $?
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
			if ! $log_info; then
				flags="-q"
			fi
			wget $src $flags
			unpack $(ls)
			cd $srcdir
			;;
		*)
			log ERROR "Unknown format!"
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
	make $MAKEFLAGS DESTDIR=$dest_dir install
}
function installpkg() { default; }

function run_part() {
	local part="$@"
	
	log INFO
	log INFO "Running $part"
	log_cmd INFO breaker
	
	set_default "${part}_func"
	failexit "Section $part failed for package $name, exiting" print_result log_cmd INFO $part
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

function create_pkginstall() {
	cat >> $tmp_dir/pkginstall.sh <<EOT
$(declare -f pre_install)

$(declare -f post_install)
EOT
}

function create_package() {
	local outfile=$1
	log INFO
	log INFO "Creating Package"
	breaker
	
	create_pkginfo
	create_pkginstall
	
	local fs_rel="fs.tar"
	local fs="$tmp_dir/$fs_rel"
	
	local manifest_rel="manifest.txt"
	local manifest="$tmp_dir/$manifest_rel"
	
	local pkg_install_rel="pkginstall.sh"
	
	local pkg_info_rel="pkginfo"
	local pkg_info="$tmp_dir/pkginfo"
	cp $name.pkginfo $pkg_info

	local result
	if str_empty $outfile; then
		result="$PWD/$name-$version.spakg"
	else
		result="$outfile"	
	fi
	
	cd $dest_dir
		log_cmd INFO tar -cvf $fs *
		find . -type f | xargs md5sum > "${manifest}"
	cd - > /dev/null
	
	log DEBUG "Creating $result from $tmp_dir"
	cd $tmp_dir
		log_cmd INFO tar -cvf $result $fs_rel $manifest_rel $pkg_install_rel $pkg_info_rel
	cd - > /dev/null
	
	log INFO
	print_result true
	log INFO
}

function setup() {
	mkdir -p $src_dir
	mkdir -p $dest_dir
}

function cleanup() {
	cd $STARTDIR
	rm -rf $tmp_dir
}

# Usage: forge file.pie
function forge() {
	local package=$1
	local outfile=$2
	
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
	
	failexit "Could not create $1 package" create_package $outfile
	
	cleanup
	
	echo $(color GREEN "$name forged successfully")
}


function usage() {
	cat <<EOT

Usage: $0 [OPTIONS] package.pie [package.spakg]
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
	local outfile
	
	for option in $@; do
		case $option in
			-p|--pretend)
				pretend=true
				;;
			-q|--quiet)
				set_log_levels ERROR
				;;
			-v|--verbose)
				set_log_levels DEBUG
				;;
			-h|--help)
				usage;;
			-t|--test)
				run_test=true;;
			*.pie)
				if file_exists $option; then
					package=$option
				else
					log ERROR "Unknown package file: $option"
					exit 1
				fi;;
			*.spakg)
				outfile="$option"
				;;
			*)
				log ERROR "Unrecognized option: $option"
				usage;;
		esac
	done
	
	if str_empty "$package"; then
		log ERROR "You must specify a package!"
		usage
	else
		forge $package $outfile
	fi
}

main $@
