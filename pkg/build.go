package main

import (
	"debug/elf"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"./libpkg"
	"github.com/pkg/errors"
)

func pkg_build(name string, opts map[harg]bool) error {
	if !opts['x'] {
		if e := opt_extract(name, opts); e != nil {
			return errors.Wrapf(e, "can not extract %v to build", name)
		}
	}

	pkg, e := libpkg.NewPkgfile(name, cfg.Ports)
	if e != nil {
		return errors.WithStack(e)
	}

	if e := pkg.Lock(); e != nil {
		return errors.Wrapf(e, "one process one time, but can not flock %v", pkg.Name)
	}
	defer pkg.Unlock()

	if e := pkg.MkPkgDir(); e != nil {
		return errors.WithStack(e)
	}

	defer func() {
		if !opts['k'] {
			pkg.RmPkgDir()
			pkg.RmSrcDir()
		}
	}()

	cmd := exec.Command("/bin/sh")
	cmd.Dir = pkg.Srcdir
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`%s
%s
%s`,
		shenv.String(),
		pkg.ShEnv(),
		pkg.Build))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Noticef("build[%v]: build()", pkg.Name)
	if e := cmd.Run(); e != nil {
		opts['k'] = true
		return errors.Wrapf(e, "can not execute build() [%v]", pkg.Name)
	}

	if e := os.Chdir(pkg.Pkgdir); e != nil {
		opts['k'] = true
		return errors.Wrapf(e, "can not cd to package dir [%v]", pkg.Name)
	}

	cpycnt := 0
	if e := filepath.Walk(pkg.Srcdir, func(path string, info os.FileInfo, err error) error {
		base := filepath.Base(path)
		if base == "COPYING" || base == "Copying" ||
			base == "COPYING.txt" || base == "Copying.txt" ||
			base == "COPYING.rst" || base == "Copying.rst" ||
			base == "LICENSE" || base == "License" ||
			base == "LICENSE.txt" || base == "License.txt" ||
			base == "LICENSE.rst" || base == "License.rst" ||
			base == "COPYRIGHT" || base == "Copyright" ||
			base == "COPYRIGHT.txt" || base == "Copyright.txt" ||
			base == "COPYRIGHT.rst" || base == "Copyright.rst" {
			licensedir := filepath.Join(pkg.Pkgdir, "share/licenses", name)
			if !exist(licensedir) {
				if e := os.MkdirAll(licensedir, 0755); e != nil {
					return e
				}
			}

			if e := movefile(path, filepath.Join(licensedir, fmt.Sprintf("LICENSE%d", cpycnt))); e != nil {
				return e
			}
			cpycnt++
		}

		return nil
	}); e != nil {
		opts['k'] = true
		return errors.Wrapf(e, "can not copy licences automatically [%v]", pkg.Name)
	}

	os.Remove("share/info/dir")
	strip := false
	if exist("/bin/strip") &&
		!pkg.NoStrip &&
		!opts['s'] {
		strip = true
	}

	if e := filepath.Walk(".", func(spath string, info os.FileInfo, err error) error {
		switch {
		case !pkg.KeepLa && strings.HasSuffix(spath, ".la"):
			if e := os.Remove(spath); e != nil {
				opts['k'] = true
				return errors.Wrapf(e, "can not remove *.la automatically [%v]", pkg.Name)
			}
		case strings.Contains(spath, "charset.alias"), strings.Contains(spath, "perllocal.pod"), strings.Contains(spath, ".packlist"):
			if e := os.Remove(spath); e != nil {
				opts['k'] = true
				return errors.Wrapf(e, "can not remove charset.alias automatically")
			}
		case !info.IsDir() && strings.HasPrefix(spath, "share/man/") && !strings.HasSuffix(spath, ".gz"):
			if info.Mode()&os.ModeSymlink != 0 {
				link, e := os.Readlink(spath)
				if e != nil {
					opts['k'] = true
					return errors.Wrapf(e, "can not follow link %v [%v]", spath, pkg.Name)
				}

				if e := os.Remove(spath); e != nil {
					opts['k'] = true
					return errors.Wrapf(e, "can not remove %v [%v]", spath, pkg.Name)
				}

				if e := os.Symlink(link+".gz", spath+".gz"); e != nil {
					opts['k'] = true
					return errors.Wrapf(e, "can not link %v to new *.gz man pages [%v]", spath, pkg.Name)
				}
			} else {
				if e := gzfile(spath, spath+".gz"); e != nil {
					opts['k'] = true
					return errors.Wrapf(e, "can not gzip %v [%v]", spath, pkg.Name)
				}

				if e := os.Remove(spath); e != nil {
					opts['k'] = true
					return errors.Wrapf(e, "can not remove old %v [%v]", spath, pkg.Name)
				}
			}
		default:
			if strip {
				if f, e := elf.Open(spath); e == nil {
					switch f.Type {
					case elf.ET_EXEC, elf.ET_DYN:
						switch f.Machine {
						case elf.EM_X86_64, elf.EM_386, elf.EM_IA_64:
							cmd = exec.Command("/bin/strip", spath)
							cmd.Dir = pkg.Pkgdir
							cmd.Stdout = os.Stdout
							cmd.Stderr = os.Stderr
							if e := cmd.Run(); e != nil {
								opts['k'] = true
								return errors.Wrapf(e, "can not strip %v [%v]", spath, pkg.Name)
							}
						}
						f.Close()
					}
				}
			}
		}
		return nil
	}); e != nil {
		return errors.Wrapf(e, "can not clean .pkg [%v]", pkg.Name)
	}

	maybetarball, e := ioutil.ReadDir(pkg.Dir)
	if e != nil {
		opts['k'] = true
		return errors.Wrapf(e, "can not read dir to clean old tarballs [%v]", pkg.Name)
	}

	for _, file := range maybetarball {
		if strings.HasPrefix(file.Name(), name+"#") && strings.HasSuffix(file.Name(), ".tgz") {
			if e := os.Remove(filepath.Join(pkg.Dir, file.Name())); e != nil {
				opts['k'] = true
				return errors.Wrapf(e, "can not remove old tarball %v [%v]", file.Name(), pkg.Name)
			}
		}
	}

	if e := pkg.PackageTarball(); e != nil {
		opts['k'] = true
		return errors.WithStack(e)
	}

	log.Noticef("build[%v]: build done", pkg.Name)
	return nil
}

func opt_build(pkg string, opts map[harg]bool) error {
	if e := locksys(); e != nil {
		return errors.WithStack(e)
	}

	var makedeps []string

	if !opts['d'] {
		var e error
		makedeps, e = pkg_deps([]string{pkg}, true, false, true, false, false)
		if e != nil {
			return errors.Wrapf(e, "can not query deps of %v", pkg)
		}
	}

	if e := pkg_install(makedeps, []string{pkg}, true, true); e != nil {
		return errors.Wrapf(e, "can not install %v", pkg)
	}

	if e := pkg_build(pkg, opts); e != nil {
		return errors.Wrapf(e, "can not build %v", pkg)
	}

	unlocksys()

	if e := opt_rmnoneed("*"); e != nil {
		return errors.Wrapf(e, "can not remove unneeded")
	}

	return nil
}
