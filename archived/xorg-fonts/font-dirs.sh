#!/bin/sh
pattern=["share/fonts"]
script='''
for i in $@; do
	mkfontscale "$i" || exit 1
	mkfontdir "$i" || exit 2
done
'''
