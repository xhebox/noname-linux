# Intro

**300++ packages, 400+ inactive packages.**

This **experimental(in the meaning of adding new packages constantly, daily using has been OK)** distro is a personal project of mine and is designed to be lightweight and fast.

You can join #nonamelinux in freenode to ask for help, if someone is there. Or leave me a mail: xw897002528@gmail.com.

# Differences from other distros

1. using native intl instead of the one in gettext
2. emoji wcwidth patch for musl
3. working fcitx
4. with some patches stolen from everywhere to let apps need EWMH(like obs) working with dwm

# Rootfs

Download the [noname.tar.gz](https://mega.nz/#F!lqpmnKxD!rWEfVB8ckIsscKFnSiy6bw)

# Important Information!
1. Run /bin/pkg as root !!!
2. use command hardlink to replace duplicate files with hardlinks!
3. Pay attention to the **ERR**, some post messages are parts of installation.
4. `export PKG_CONFIG=` to override the default config file path `/etc/pkg.conf`.
5. some laptop is affected by this bug(including me): https://bugzilla.kernel.org/show_bug.cgi?id=156341
6. ```tpm_crb MSFT0101:00: can't request region for resource```, if you see it in demsg, be calm, tpm2.0 is not fully working.
