#!/bin/bash

function usage() {
	cat <<EOT
Usage: $0
	check
	forge, build
	wield, install
	search
	info
	packages
	news
	audit
	-h, --help (this menu)
EOT
}

function main() {
	local option
	local package
	while option="$1" && [ "$option" != "" ] && shift 1; do
		case $option in
			check)
				;;
			-h|--help)
				usage
				exit 0
				;;
			forge|build)
				
				;;
			wield|install)
			
				;;
			search)
			
				;;
			*)
				usage
				exit 1
		esac
	done
}


main $@
