#!/bin/bash

rpcFile="./internal/static/rpc.ts";

data="$(cat "$rpcFile")";
(
	while read line; do
		if [ -z "$line" ]; then
			echo -en "\nconst";
			num=0;
			while read c; do
				if [ $num -gt 0 ]; then
					echo -n ",";
				fi;
				let "num++";
				echo -n " $c = -$num";
			done < <(sed -n '/^const ($/{:loop n;/^)$/q; p; b loop}' ./rpc.go | grep -v "^$" | sed -e 's/^	//' -e 's/ .*//');
			echo -e ";\n\n";
			break;
		fi;
		echo "$line";
	done;
	cat - | grep -v "const broadcast";
) < <(echo "$data") | awk -v RS='\n\n\n' 1 > "$rpcFile";
