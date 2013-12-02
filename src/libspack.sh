#!/bin/bash

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
			r_color="0;33";;
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
				if is_integer 1; then
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
	if $log_info; then
		color BROWN $(printf %$(tput cols)s | tr " " "=")
	fi
}

function print_result() {
	$@
	if [ $? -eq 0 ]; then
		log_cmd INFO echo $(color GREEN "Success")
	else
		log_cmd INFO echo $(color RED "Error")
	fi
}

function is_integer() {
	[[ $1 =~ ^-?[0-9]+$ ]]
}
