#!/bin/sh
if [ "$HOME" != "" ]; then
	if [ -e "$HOME/.stack/env" ]; then
		. "$HOME/.stack/env"
	fi
fi
/lib/stack $@
