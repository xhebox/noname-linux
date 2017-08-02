#!/bin/sh
if grep -q -F "lib/graphviz" "${1}/${2}.footprint"; then
	if [ -e /bin/dot ]; then
		rm -f "$pkgdir"/lib/graphviz/config
		rm -f "$pkgdir"/lib/graphviz/config6
		/bin/dot -c
		if [ "$?" -ne 0 ]; then
			echo "$2: Failed to update graphviz plugins!"
			return 1
		fi
		echo 'Succeeded to update graphviz plugins'
	fi
fi
