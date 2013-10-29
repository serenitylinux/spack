#!/bin/bash

source ../lib.sh

function usage() {
	cat <<EOT

Usage: $0 [OPTIONS] package.spakg
	-p, --pretend
	-v, --verbose
	-q, --quiet
	-h, --help

EOT
exit 1
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
			*.spkg)
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
		
		install_pkg $package
	done
}

main $@
