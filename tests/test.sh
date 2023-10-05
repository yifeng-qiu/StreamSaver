#!/bin/sh

if [ $# < 2 ] 
then
	cat << EOF
usage: $0 [post | get] DATA

Examples:
	$0 post			Post to the server
	$0 get 10000	Retrieve random ID 10000 times
EOF
	exit 0	
fi

if [ $1 == "post" -o $1 == "POST" ]
then
# POST random text from man pages
	man curl | egrep -o '[a-zA-Z]{3,8}' | xargs -I % curl -iX POST localhost:1718 -F text=%
elif [ $1 == "get" -o $1 == "GET" ]
then
# GET random ID from webserver

	RANDOM=$$
	for i in `seq 10000`
	do
		curl -iX GET localhost:1718 -d "{\"id\":$RANDOM}"
	done
fi
