#!/bin/bash

source /usr/lib/spack/libspack

function refresh_repos() {
	for i in /etc/spack/repos/*; do
		ls $i
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

function main() {
	local package
	local file
	case $1 in
		refresh)
			refresh_repos
			exit -1
			;;
		upgrade)
			log ERROR "upgrade option not implemented"
			exit -1
		-h|--help)
			usage
			exit 0
			;;
		forge|build)
			case $2 in
				-f|--file)
					file="$3"
				;;
				*.pie)
					file="$2"
				;;
				*)
					package="$2"
					file=$(get_pie $package)
				;;
			esac
			
			forge $file

			exit 0
			;;
		wield|install)
			case $2 in
				-f|--file)
					file="$3"
				;;
				*.spakg)
					file="$2"
				;;
				*)
					#search repo and get package[s]
					#file=$(get_spakg $2)
					log ERROR "installation from repo not supported"
					exit 1
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
