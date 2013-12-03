#!/bin/bash

source /etc/spack/spack.conf

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
			r_color="1;31";;
		GREEN|green)
			r_color="1;32";;
		BROWN|brown)
			r_color="0;33";;
		BLUE|blue)
			r_color="1;34";;
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

function set_log_level() {
	local level="$1"
	local value="$2"
	case $level in
		DEBUG|debug)
			log_debug=$value
			;;
		INFO|info)
			log_info=$value
			;;
		WARN|warn)
			log_=$value
			;;
		ERROR|error)
			log_error=$value
			;;
		*)
			color RED "Invalid Log Level: $level"; echo
			exit -1
			;;
	esac
}

function set_log_levels() {
	local level="$1"
	set_log_level DEBUG false
	set_log_level INFO false
	set_log_level WARN false
	set_log_level ERROR false

	case $level in
		DEBUG|debug)
			set_log_level DEBUG true
			;&
		INFO|info)
			set_log_level INFO true
			;&
		WARN|warn)
			set_log_level WARN true
			;&
		ERROR|error)
			set_log_level ERROR true
			;;
		*)
			color RED "Invalid Log Level: $level"; echo
			exit -1
			;;
	esac
}

set_log_levels INFO

# Usage: log $level $string
function log() {
	local level="$1"
	shift
	local string="$@"
	case $level in
		INFO)
			if $log_info; then
				color WHITE $@
				echo
			fi;;
		DEBUG)
			if $log_debug; then
				color BLUE $@
				echo
			fi;;
		WARN)
			if $log_warn; then
				color YELLOW "Warning: "
				echo $@
			fi;;
		ERROR)
			if $log_error; then
				if is_integer $1; then
					color RED "ERROR $1: "
					shift
				else
					color RED "ERROR: "
				fi
				echo $@ 
			fi;;
	esac
}

function log_cmd() {
	local printme
	local level="$1"
	shift
	case $level in
		INFO)
			printme=$log_info;;
		DEBUG)
			printme=$log_debug;;
		WARN)
			printme=$log_warn;;
		ERROR)
			printme=$log_error;;
	esac
	
	if $printme; then
		$@
	else
		$@ > /dev/null
	fi
}

function breaker() {
	log_cmd INFO color BROWN $(printf %$(tput cols)s | tr " " "=")
}

function print_result() {
	$@
	if [ $? -eq 0 ]; then
		log INFO
		log_cmd INFO echo $(color GREEN "Success")
	else
		log ERROR
		log_cmd ERROR echo $(color RED "Error")
	fi
}

function is_integer() {
	[[ $1 =~ ^-?[0-9]+$ ]]
}


function gracefull_failure() {
	trap exit EXIT
	local code="$?"
	local message="$@"
	set +e
	echo
	log ERROR $code $message
	cleanup
	exit 1
}

function failexit() {
	local message="$1"
	shift
	local func="$@"
	
	set -e
	trap "gracefull_failure $message" EXIT
	
	$func
	
	trap exit EXIT
	set +e
}

function require_root() {
	if [[ $EUID -ne 0 ]]; then
		log ERROR "You must be root to run this function, try sudo $0"
		exit -1
	fi
}

function func_exists() {
	declare -f $1 > /dev/null
}
