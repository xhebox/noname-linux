#!/bin/sh
if [ "$HOME" != "" ]; then
	if [ -e "$HOME/.stack/proxy" ]; then
		. "$HOME/.stack/proxy"
	fi
fi
/lib/stack $@
