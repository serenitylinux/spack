#!/bin/bash

source /usr/lib/spack/libspack
set -e

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
