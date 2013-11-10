#!/bin/bash

# Usage: failexit critial_command args
function failexit() {
	$@
	if [ $? -ne 0 ]; then 
		log ERROR "$1 failed, exiting"
		exit -1
	fi
}

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
			$log_info && color WHITE $@ && echo ;;
		DEBUG)
			$log_debug && color BLUE $@ && echo ;;
		WARN)
			$log_warn && color YELLOW "warning: " && echo $@ ;;
		ERROR)
			$log_error && color RED "error: " && echo $@ ;;
	esac
}
