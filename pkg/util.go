package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"
)

var (
	syslock = flock.New(filepath.Join(os.TempDir(), "pkg.lock"))
)

// util functions
func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func ask(format string, args ...interface{}) bool {
	act := "no"
	fmt.Printf(format, args...)
	fmt.Scanf("%s", &act)
	switch act {
	case "no", "n", "N":
		return false
	default:
		return true
	}
}

func exist(path string) bool {
	_, e := os.Stat(path)
	return !os.IsNotExist(e)
}

func movefile(src, dst string) error {
	return os.Rename(src, dst)
}

func gzfile(src, dst string) error {
	in, e := os.Open(src)
	if e != nil {
		return errors.Wrapf(e, "can not compress[%v -> %v]", src, dst)
	}
	defer in.Close()

	info, e := in.Stat()
	if e != nil {
		return errors.Wrapf(e, "can not compress[%v -> %v]", src, dst)
	}

	out, e := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, info.Mode())
	if e != nil {
		return errors.Wrapf(e, "can not compress[%v -> %v]", src, dst)
	}
	defer out.Close()

	gzwt := gzip.NewWriter(out)
	defer gzwt.Close()

	if _, e := io.Copy(gzwt, in); e != nil {
		return errors.Wrapf(e, "can not compress[%v -> %v]", src, dst)
	}

	return nil
}

func locksys() error {
	ok, e := syslock.TryLock()

	if !ok {
		return errors.WithStack(e)
	}

	return nil
}

func unlocksys() error {
	if e := syslock.Close(); e != nil {
		return errors.WithStack(e)
	}

	return nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
