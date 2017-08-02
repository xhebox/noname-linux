#!/bin/sh
set -e
[ -d /lib/sysusers.d ] && list=`$list ls -1 /lib/sysusers.d`
[ -d /etc/sysusers.d ] && list=`$list ls -1 /etc/sysusers.d`
[ -d /run/sysusers.d ] && list=`$list ls -1 /run/sysusers.d`
for i in $list; do
	sysusers "`basename \"$i\"`"
done
