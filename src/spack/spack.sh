#!/bin/bash

source /usr/lib/spack/libspack

function usage() {
	cat <<EOT
Usage: $0
	check
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

function main() {
	local package
	local file
	case $1 in
		check)
			log ERROR "Check option not implemented"
			exit -1
			;;
		-h|--help)
			usage
			exit 0
			;;
		forge|build)
			case $2 in
				-f|--file)
					file="$3"
				;;
				*)
					package="$2"
					file=$(get_package $package)
				;;
			esac
			
			forge $file

			exit 0
			;;
		wield|install)
			case $2 in
				-f|--file)
					wield $3
				;;
				*)
					#search repo and get package[s]
					package="$2"
					log ERROR "installation from repo not supported"
				;;
			esac
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
