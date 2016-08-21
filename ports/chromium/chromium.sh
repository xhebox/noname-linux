#!/bin/sh
export CHROME_WRAPPER=/lib/chromium/chromium
export CHROME_DESKTOP=chromium.desktop
exec /lib/chromium/chromium $CHROME_FLAGS "$@"
