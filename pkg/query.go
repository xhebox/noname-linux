package main

import (
	"fmt"
	"path/filepath"

	"./libpkg"
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

func opt_query(pkg string, opts map[harg]bool) error {
	if opts['a'] {
		for v := range db.Keys(nil) {
			fmt.Println(v)
		}
	} else if opts['d'] {
		fmt.Println(pkg)
		pkgs, e := pkg_required([]string{pkg})
		if e != nil {
			return e
		}

		for _, v := range pkgs {
			fmt.Printf("\t%v\n", v)
		}
	} else if opts['p'] {
		glob := glob.MustCompile(pkg)
		for v := range db.Keys(nil) {
			if glob.Match(v) {
				fmt.Println(v)
			}
		}
	} else if opts['o'] {
		for x := range db.Keys(nil) {
			pkgdb, e := libpkg.NewPkgdb(x, db)
			if e != nil {
				return errors.Errorf("%v is not installed properly", x)
			}

			for _, fp := range pkgdb.Footprint {
				if r, _ := filepath.Rel(filepath.Join(cfg.ROOT, pkg), filepath.Join(cfg.ROOT, fp.Path)); r == "." {
					log.Noticef("opt_query[%v]: matched[%v]", pkg, x)
					return nil
				}
			}
		}
	} else {
		pkgdb, e := libpkg.NewPkgdb(pkg, db)
		if e != nil {
			return errors.Errorf("%v is not installed properly", pkg)
		}

		if opts['l'] {
			for _, fp := range pkgdb.Footprint {
				fmt.Printf("%+v\n", fp)
			}
		} else {
			fmt.Printf(`name: %v
version: %v
desc: %v
baks: %v
deps: %v
user: %v
`,
				pkgdb.Name,
				pkgdb.Version,
				pkgdb.Desc,
				pkgdb.Baks,
				pkgdb.Deps,
				pkgdb.User)
		}
	}

	return nil
}
