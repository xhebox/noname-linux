package main

import (
	"archive/tar"
	"io/ioutil"
	"os"
	"path/filepath"

	"./libpkg"
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"v.io/x/lib/toposort"
)

func pkg_required(name []string) ([]string, error) {
	sorter := toposort.Sorter{}

	current := append([]string{}, name...)
	newcur := []string{}

	var pkgdbs []struct {
		name string
		deps map[string]bool
	}

	for _, v := range current {
		if !db.Has(v) {
			return nil, errors.Errorf("%s is not installed", v)
		}
	}

	for v := range db.Keys(nil) {
		pkgdb, e := libpkg.NewPkgdb(v, db)
		if e != nil {
			return nil, errors.WithStack(e)
		}

		t := map[string]bool{}

		for _, x := range pkgdb.Deps {
			t[x] = true
		}

		pkgdbs = append(pkgdbs, struct {
			name string
			deps map[string]bool
		}{v, t})
	}

	for {
		for _, cur := range current {
			sorter.AddNode(cur)
		}

		for _, pkgdb := range pkgdbs {
			t := pkgdb.deps

			for _, x := range current {
				if t[x] {
					newcur = append(newcur, pkgdb.name)
					sorter.AddEdge(x, pkgdb.name)
					break
				}
			}
		}

		_, cycles := sorter.Sort()
		if len(cycles) > 0 {
			return nil, errors.Errorf("detect circular deps%v", cycles)
		}

		if len(newcur) == 0 {
			break
		}

		oldcur := current
		current = newcur
		newcur = oldcur[0:0]
	}

	r := []string{}

	v, _ := sorter.Sort()
	for _, val := range v {
		r = append(r, val.(string))
	}

	log.Debugf("pkg_required[%v]: queried requiredby[%v]\n", name, r)
	return r, nil
}

func pkg_rmfiles(files []libpkg.Fileinfo, baks []string) error {
	for _, x := range files {
		if x.Type != tar.TypeDir {
			os.Remove(filepath.Join(cfg.ROOT, x.Path))
		}
	}

	for _, x := range files {
		if x.Type == tar.TypeDir {
			p, e := ioutil.ReadDir(x.Path)
			if e == nil && len(p) != 0 {
				os.Remove(filepath.Join(cfg.ROOT, x.Path))
			}
		}
	}

	for _, x := range baks {
		os.Remove(filepath.Join(cfg.ROOT, x+".bak"))
	}

	return nil
}

func pkg_remove(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}

	var rfiles []libpkg.Fileinfo
	var rbaks []string

	for _, pkg := range pkgs {
		pkgdb, e := libpkg.NewPkgdb(pkg, db)
		if e != nil {
			return errors.WithStack(e)
		}

		rfiles = append(rfiles, pkgdb.Footprint...)
		rbaks = append(rbaks, pkgdb.Baks...)
	}

	if e := pkg_hook(rfiles, "pre", "remove"); e != nil {
		return e
	}

	if e := pkg_rmfiles(rfiles, rbaks); e != nil {
		return errors.Wrapf(e, "can not remove files")
	}

	if e := pkg_hook(rfiles, "post", "remove"); e != nil {
		return e
	}

	log.Noticef("remove%v: removed files", pkgs)

	for _, pkg := range pkgs {
		if e := libpkg.Erase(pkg, db); e != nil {
			return errors.WithStack(e)
		}
	}

	return nil
}

func opt_remove(pkgs []string, opts map[harg]bool) error {
	if e := locksys(); e != nil {
		return errors.WithStack(e)
	}
	defer unlocksys()

	if !opts['d'] {
		var e error
		pkgs, e = pkg_required(pkgs)
		if e != nil {
			return errors.Wrapf(e, "can not query required of %v", pkgs)
		}
	}

	if len(pkgs) > 1 {
		if !ask("do you really want to remove %v\n", pkgs) {
			return nil
		}
	}

	return pkg_remove(pkgs)
}

func opt_rmnoneed(pattern string) error {
	if e := locksys(); e != nil {
		return errors.WithStack(e)
	}
	defer unlocksys()

	var pkgdbs []*libpkg.Pkgdb

	m := map[string][]string{}

	for v := range db.Keys(nil) {
		pkgdb, e := libpkg.NewPkgdb(v, db)
		if e != nil {
			return errors.WithStack(e)
		}

		for _, x := range pkgdb.Deps {
			m[x] = append(m[x], x)
		}

		pkgdbs = append(pkgdbs, pkgdb)
	}

	var rmlist []string
	t := map[string]bool{}

	glob := glob.MustCompile(pattern)

	for l, n := -1, 0; l != n; l, n = n, len(rmlist) {
		for _, v := range pkgdbs {
			if glob.Match(v.Name) && !t[v.Name] && len(m[v.Name]) == 0 && !v.User {
				rmlist = append(rmlist, v.Name)
				t[v.Name] = true

				newmx := []string{}
				for _, x := range m[v.Name] {
					if x != v.Name {
						newmx = append(newmx, x)
					}
				}
				m[v.Name] = newmx
			}
		}
	}

	reverse(rmlist)

	if len(rmlist) > 1 {
		if !ask("do you really want to remove %v\n", rmlist) {
			return nil
		}
	}

	if e := pkg_remove(rmlist); e != nil {
		return errors.Wrapf(e, "can not remove files")
	}

	return nil
}
