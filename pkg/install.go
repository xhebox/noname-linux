package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	libpkg "./pkg"
	"github.com/pkg/errors"
	"v.io/x/lib/toposort"
)

func pkg_deps(name []string, makedep, fdb, installed, needed, user bool) ([]string, error) {
	// to avoid duplication quickly
	m := map[string]bool{}
	sorter := toposort.Sorter{}

	current := append([]string{}, name...)
	newcur := []string{}

	for {
		for _, cur := range current {
			if m[cur] {
				continue
			}

			m[cur] = true

			sorter.AddNode(cur)

			if user {
				pkg, e := libpkg.NewPkgdb(cur, db)
				if e == nil && pkg.User {
					continue
				}
			}

			var deps []string
			pkg, e := libpkg.NewPkgfile(cur, cfg.Ports)
			if e != nil {
				return nil, errors.WithStack(e)
			}

			if makedep {
				deps = pkg.Makedeps
				// when we ask for makedeps, we generally want makedeps and their deps, instead of makedeps and their makedeps
				// so, disable the flag after adding the first-level makedeps
				makedep = false
			} else {
				deps = pkg.Deps
			}

			// kick packages needed by others
			if needed {
				t := map[string]bool{}

				for v := range db.Keys(nil) {
					pkgdep, e := libpkg.NewPkgdb(v, db)
					if e != nil {
						return nil, errors.WithStack(e)
					}

					for _, v := range pkgdep.Deps {
						t[v] = true
					}
				}

				newdeps := []string{}
				for _, v := range deps {
					if !t[v] {
						newdeps = append(newdeps, v)
					}
				}
				deps = newdeps
			}

			for _, v := range deps {
				if installed && db.Has(v) {
					continue
				}

				newcur = append(newcur, v)
				sorter.AddEdge(cur, v)
			}
		}

		_, cycles := sorter.Sort()
		if len(cycles) > 0 {
			return nil, errors.Errorf("detect circular deps%v", cycles)
		}

		// no new deps anymore, we've visited every corner
		if len(newcur) == 0 {
			break
		}

		// exchange two variables to reuse
		oldcur := current
		current = newcur
		newcur = oldcur[0:0]
	}

	r := []string{}

	v, _ := sorter.Sort()
	for _, val := range v {
		r = append(r, val.(string))
	}

	log.Debugf("pkg_deps: queried deps%v", r)
	return r, nil
}

func pkg_cpfiles(pkg *libpkg.Pkgfile) error {
	fd, e := os.Open(pkg.Tarball)
	if e != nil {
		return errors.Wrapf(e, "can not open tarball to extract [%v]", pkg.Tarball)
	}
	defer fd.Close()

	gzfd, e := gzip.NewReader(fd)
	if e != nil {
		return errors.Wrapf(e, "can not ungzip tarball to extract [%v]", pkg.Tarball)
	}
	defer gzfd.Close()

	tarfd := tar.NewReader(gzfd)
	for {
		hdr, e := tarfd.Next()
		if e == io.EOF {
			break
		} else if e != nil {
			return errors.Wrapf(e, "can not untar tarball to extract [%v]", pkg.Tarball)
		}

		// backup files
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo, tar.TypeLink, tar.TypeSymlink:
			for _, v := range pkg.Baks {
				if v == hdr.Name && exist(filepath.Join(cfg.ROOT, v)) {
					hdr.Name += ".bak"
				}
			}
		}

		dst := filepath.Join(cfg.ROOT, hdr.Name)
		mode := os.FileMode(hdr.Mode)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if e := os.MkdirAll(dst, mode); e != nil {
				return errors.Wrapf(e, "can not install dir %v [%v]", dst, pkg.Tarball)
			}
		case tar.TypeReg, tar.TypeRegA:
			dstfd, e := os.OpenFile(dst+".tmp", os.O_RDWR|os.O_CREATE, mode)
			if e != nil {
				return errors.Wrapf(e, "can not install tmp file %v [%v]", dst+".tmp", pkg.Tarball)
			}

			_, e = io.Copy(dstfd, tarfd)
			if e != nil {
				return errors.Wrapf(e, "can not fill tmp file %v [%v]", dst+".tmp", pkg.Tarball)
			}
			dstfd.Close()

			if e := movefile(dst+".tmp", dst); e != nil {
				return errors.Wrapf(e, "can not overwrite the origin file %v [%v]", dst, pkg.Tarball)
			}
		case tar.TypeSymlink:
			os.Remove(dst)
			if e := os.Symlink(hdr.Linkname, dst); e != nil {
				return errors.Wrapf(e, "can not overwrite the soft link %v [%v]", dst, pkg.Tarball)
			}
		case tar.TypeLink:
			if e := os.Link(hdr.Linkname, dst); e != nil {
				return errors.Wrapf(e, "can not overwrite the hard link %v [%v]", dst, pkg.Tarball)
			}
		default:
			return errors.Errorf("%s: unsupported type flag: %c [%v]", hdr.Name, hdr.Typeflag, pkg.Tarball)
		}
	}

	log.Noticef("install[%v]: extracted files", pkg.Name)
	return nil
}

func pkg_install(pkgs []string, main []string, nomain, user bool) error {
	var pkgfiles []*libpkg.Pkgfile
	var pkgdbs []*libpkg.Pkgdb
	var footprints [][]libpkg.Fileinfo
	var rfiles []libpkg.Fileinfo
	m := map[string]bool{}

	for _, v := range main {
		m[v] = true
	}

	for _, pkg := range pkgs {
		if nomain && m[pkg] {
			continue
		}

		pkgfile, e := libpkg.NewPkgfile(pkg, cfg.Ports)
		if e != nil {
			return errors.WithStack(e)
		}

		// check if tarball exists
		if !exist(pkgfile.Tarball) {
			return errors.Errorf("can not find tarball %s [%v]", pkgfile.Tarball, pkgfile.Name)
		}

		// generate footprint from tarball
		footprint, e := pkgfile.Footprint()
		if e != nil {
			return errors.WithStack(e)
		}

		pkgfiles = append(pkgfiles, pkgfile)
		footprints = append(footprints, footprint)
		rfiles = append(rfiles, footprint...)
	}

	if e := pkg_hook(rfiles, "pre", "install"); e != nil {
		return e
	}

	for k, pkgfile := range pkgfiles {
		dbuser := false

		if m[pkgfile.Name] {
			dbuser = user
		}

		if db.Has(pkgfile.Name) {
			pkgdb, e := libpkg.NewPkgdb(pkgfile.Name, db)
			if e != nil {
				return errors.WithStack(e)
			}

			if pkgdb.User {
				dbuser = true
			}

			t := map[string]bool{}

			for _, x := range footprints[k] {
				t[x.Path] = true
			}

			for _, x := range pkgdb.Footprint {
				if !t[x.Path] {
					os.Remove(filepath.Join(cfg.ROOT, x.Path))
				}
			}
		}

		pkgdbs = append(pkgdbs, &libpkg.Pkgdb{
			Footprint: footprints[k],
			Name:      pkgfile.Name,
			Version:   pkgfile.Version,
			Deps:      pkgfile.Deps,
			Desc:      pkgfile.Desc,
			Baks:      pkgfile.Baks,
			User:      dbuser,
		})

		if e := pkg_cpfiles(pkgfile); e != nil {
			return errors.WithStack(e)
		}
	}

	if e := pkg_hook(rfiles, "post", "install"); e != nil {
		return e
	}

	for _, pkgdb := range pkgdbs {
		if e := pkgdb.Store(db); e != nil {
			return errors.WithStack(e)
		}
		log.Noticef("install[%v]: updated db", pkgdb.Name)
	}

	return nil
}

func opt_install(pkgs []string, opts map[harg]bool) error {
	if e := locksys(); e != nil {
		return errors.WithStack(e)
	}
	defer unlocksys()

	var all []string

	if !opts['d'] {
		var e error
		all, e = pkg_deps(pkgs, false, false, !opts['r'], false, false)
		if e != nil {
			return errors.Wrapf(e, "can not query deps [%v]", pkgs)
		}
	}

	if len(all) > 1 {
		if !ask("do you really want to install %v\n", all) {
			return nil
		}
	}

	return pkg_install(all, pkgs, false, true)
}
