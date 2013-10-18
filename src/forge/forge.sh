#!/bin/sh

#Globals
pretend=false
verbose=false

# TODO move to lib
# Usage: debug $string
function debug() {
	$verbose && echo $1
}


# Usage: forge file.pie
function forge() {
	local package=$1
	echo "Forging package $1 in the heart of a star."
	debug "This can be a dangerous operation, please read the instruction manual to prevent a black hole."
	$pretend && echo "Just kidding :P"
}


function usage() {
	cat <<EOT

Usage: $0 [OPTIONS] package.pie
	-p, --pretend
	-v, --verbose
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
			-v|--verbose)
				verbose=true;;
			-h|--help)
				echo "Usage!"
				usage;;
			*.pie)
				if [ -f $option ]; then
					package=$option
				else
					echo "Unknown package file: $option"
					exit 1
				fi;;
			*)
				echo "Unrecognized option: $option"
				usage;;
		esac
	done
	
	forge $package
}





main $@
