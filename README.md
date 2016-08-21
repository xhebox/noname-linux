# Intro

**600++ packages now!**

This **experimental** distro is a personal project of mine and is designed to be lightweight and fast.

**arch: x86_64**

**kernel: using zen and uskm**

**libc: musl(not glibc, be careful)**

**bootloader: grub(syslinux is in the ports)**

**init: finit**

**shell: default to dash, but bash is included**

You can join #nonamelinux in freenode to ask for help, if someone is there. Or leave me a mail: xw897002528@gmail.com.

# Differences from other distros

1. using native intl instead of the one in gettext
2. emoji wcwidth patch for musl
3. usable wine but not perfect
4. working blender
5. working fcitx
6. with some patches stolen from everywhere to let apps need EWMH(like obs) working with dwm

# Installation

### boot to livecd
Download the [noname.iso](https://my.pcloud.com/publink/show?code=kZXEMTZpLoGyohz5bS7XnjSbwD2kSVYBUwV).

Feel free to burn it into usbdisk or real CD. `dd if= of= bs=1M` should be enough.

### prepare your partition
I recommend cfdisk command.

After it, mount your partition to /mnt(also other partitions to /mnt/xxx if needed). README is stored in iso for reading while installing(/README.md).

### about network in livecd
ifconfig is deprecated.
`ip link set up dev $name` should work for activing the interface.
wpa_supplicant is included for wireless networks.
lynx text browser is included in livecd, either.

### start installation
run the following to install extactly same packages in livecd to /mnt:
```
install-rootfs /mnt(or else target dir)
```

### configure your installation
use the following to chroot in /mnt(/mnt/sys, /mnt/proc, /mnt/dev is mounted with -o bind automatically):
```
nchroot /mnt
```

Config files you need to edit:
+ filesystem
/etc/fstab(lsblk -f, if you want to use UUID or LABEL in fstab)
+ finit
/etc/rc.local(post init scripts)
/etc/finit.conf(the init config file)
+ grub
/etc/default/grub(if you need to add resume swap partition)
+ acpid
/etc/acpi/handler.sh(acpid is enable by default to handle acpi events)

**root passwd is empty by default**

Optional config: add users, change root passwd, select the correct timezone or LANG...Reading Gentoo and Arch's wiki is a good choice.

### make it bootable
run: grub-install --target=i386-pc /dev/sdx
     grub-mkconfig -o /boot/grub/grub.cfg

To UEFI systems, efivarfs is not mounted by default(livecd mounted it, so that you can install UEFI bootloaders), here's the detailed [description from archlinux](https://wiki.archlinux.org/index.php/Unified_Extensible_Firmware_Interface#Mount_efivarfs)

### reboot and get the package manager
Congratulations, enjoy it! But dont forget to copy ./pkg to /bin/pkg, ./ignore to /etc/pkg.ignore, ./config to /etc/pkg.conf, now you have a working package manager.

### about kernel
1. there's two packages for kernel, one is linux-x86_64, another one is linux.
linux-x86_64 is compiled with `march=x86-64` while linux is compiled with `march=native`. linux-x86_64 is installed by default, but you may prefer the latter as it's a lot faster.
2. ```tpm_crb MSFT0101:00: can't request region for resource```, if you see it in demsg, be calm, tpm2.0 is not fully working.


# Important Information!
1. Run /bin/pkg as root !!!
2. use command hardlink to replace duplicate files with hardlinks!
3. Pay attention to the **ERR**, some post messages are parts of installation.
4. `export PKG_CONFIG=` to override the default config file `/etc/pkg.conf`.
