#!/bin/sh

# At this time, while this script is trivial, we ignore any parameters given.
# However, for backwards compatibility reasons, future versions of this script must
# support the syntax "update-ca-trust extract" trigger the generation of output
# files in $DEST.

DEST=/etc/ca-certificates/extracted

# Prevent p11-kit from reading user configuration files.
export P11_KIT_NO_USER_CONFIG=1

extract() {
  trust extract --overwrite "$@"
}

## Simple PEM bundles
extract --comment --format=pem-bundle --filter=ca-anchors --purpose=server-auth  $DEST/tls-ca-bundle.pem
extract --comment --format=pem-bundle --filter=ca-anchors --purpose=email        $DEST/email-ca-bundle.pem
extract --comment --format=pem-bundle --filter=ca-anchors --purpose=code-signing $DEST/objsign-ca-bundle.pem

## OpenSSL PEM bundle that includes trust flags
extract --comment --format=openssl-bundle --filter=certificates $DEST/ca-bundle.trust.crt

## TianoCore EDK II bundle
extract --format=edk2-cacerts --filter=ca-anchors --purpose=server-auth $DEST/edk2-cacerts.bin

## Java bundle
extract --format=java-cacerts --filter=ca-anchors --purpose=server-auth /etc/ssl/certs/java/cacerts

## OpenSSL-style directory with individual PEM files and hash links
# The directory-format extractors remove all files in the target directory, but not directories or files therein
extract --format=pem-directory-hash --filter=ca-anchors --purpose=server-auth $DEST/cadir

# We don't want to have to remove everything from the certs directory but neither
# do we want to leave stale certs around, so only place symlinks in the real cadir
for f in $DEST/cadir/*; do
  ln -fsr -t /etc/ssl/certs "$f"
done

# Now find and remove all broken symlinks
find -L /etc/ssl/certs -maxdepth 1 -type l -delete
