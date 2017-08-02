package main

import (
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func chroot_setup(root string) error {
	if filepath.Clean(root) == "/" {
		return nil
	}

	if e := unix.Mount("proc", filepath.Join(root, "proc"), "proc", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, ""); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("sys", filepath.Join(root, "sys"), "sysfs", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, ""); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("udev", filepath.Join(root, "dev"), "devtmpfs", unix.MS_NOSUID, "mode=0755"); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("devpts", filepath.Join(root, "dev/pts"), "devpts", unix.MS_NOSUID|unix.MS_NOEXEC, "mode=0620,gid=5"); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("shm", filepath.Join(root, "dev/shm"), "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "mode=1777"); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("run", filepath.Join(root, "run"), "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "mode=0755"); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if e := unix.Mount("tmp", filepath.Join(root, "tmp"), "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_STRICTATIME, "mode=1777"); e != nil {
		return errors.Wrapf(e, "can not setup chroot")
	}

	if exist(filepath.Join(root, "sys/firmware/efi/efivars")) {
		if e := unix.Mount("efivarfs", filepath.Join(root, "sys/firmware/efi/efivars"), "efivarfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); e != nil {
			return errors.Wrapf(e, "can not setup chroot")
		}
	}

	return nil
}

func chroot_unsetup(root string) error {
	if filepath.Clean(root) == "/" {
		return nil
	}

	if exist(filepath.Join(root, "sys/firmware/efi/efivars")) {
		if e := unix.Unmount(filepath.Join(root, "sys/firmware/efi/efivars"), 0); e != nil {
			return errors.Wrapf(e, "can not unsetup chroot")
		}
	}

	if e := unix.Unmount(filepath.Join(root, "tmp"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "run"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "dev/shm"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "dev/pts"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "dev"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "sys"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	if e := unix.Unmount(filepath.Join(root, "proc"), 0); e != nil {
		return errors.Wrapf(e, "can not unsetup chroot")
	}

	return nil
}
