#!/bin/sh
if grep -q -F "immodules" "${1}/${2}.footprint"; then
	if [ -e /bin/gtk-query-immodules-2.0 ]; then
		/bin/gtk-query-immodules-2.0 --update-cache
		if [ "$?" -ne 0 ]; then
			echo "$2: Failed to update gtk2 immodules cache!"
			return 1
		fi
		echo 'Succeeded to update gtk2 immodules cache'
	fi
fi
