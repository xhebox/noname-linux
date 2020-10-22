## Intro

**300++ packages, 500+ inactive packages.**

This **experimental(in the meaning of adding/removeing packages constantly)** distro is a personal project of mine and is designed to be lightweight and fast.

Leave me a mail if you have any questions: xw897002528@gmail.com.

## Differences from other distros

1. using native intl instead of the one in gettext
2. emoji wcwidth patch for musl

## Important Information

1. Run /bin/pkg as root !!!
2. use command hardlink to replace duplicate files with hardlinks!
3. Pay attention to the **ERR**, some post messages are parts of installation.
4. `export PKG_CONFIG=` to override the default config file path `/etc/pkg.conf`.
5. ```tpm_crb MSFT0101:00: can't request region for resource```, if you see it in demsg, be calm, tpm2.0 is not fully working.
