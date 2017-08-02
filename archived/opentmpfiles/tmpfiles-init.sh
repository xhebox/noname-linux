#!/bin/sh
set -e
tmpfiles --prefix=/dev --create --boot
tmpfiles --exclude-prefix=/dev --create --remove --boot
