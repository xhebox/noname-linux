#!/bin/sh
unset GDK_BACKEND
export CHROME_WRAPPER=/lib/chromium/chromium
export CHROME_DESKTOP=chromium.desktop
export DBUS_SESSION_BUS_ADDRESS="unix:path=$XDG_RUNTIME_DIR/bus"
exec /lib/chromium/chromium $CHROME_FLAGS "$@"
