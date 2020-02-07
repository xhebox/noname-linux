package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobwas/glob"
	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type hookfile struct {
	Pattern   []string `toml:"pattern"`
	Hook      string   `toml:"script"`
	NoArg     bool     `toml:"noarg"`
	When      []string `toml:"when"`
	Operation []string `toml:"operation"`
	pattern   []glob.Glob
}

type byString []os.FileInfo

func (s byString) Len() int {
	return len(s)
}

func (s byString) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byString) Less(i, j int) bool {
	return s[i].Name() < s[j].Name()
}

func pkg_hook(files []Fileinfo, when, opt string) error {
	log.Debugf("pkg_hook: %s, %s", when, opt)

	if len(files) == 0 {
		return nil
	}

	hooksdir, e := ioutil.ReadDir(cfg.Hooks)
	if e != nil {
		return nil
	}

	newhooksdir := byString(hooksdir)
	sort.Sort(newhooksdir)

	if e := chroot_setup(cfg.ROOT); e != nil {
		return errors.WithStack(e)
	}
	defer func() {
		if e := chroot_unsetup(cfg.ROOT); e != nil {
			log.Errorf("%+v", errors.Wrapf(e, "can not unsetup chroot"))
		}
	}()

	for _, script := range newhooksdir {
		h := hookfile{}

		contents, e := ioutil.ReadFile(filepath.Join(cfg.Hooks, script.Name()))
		if e != nil {
			return errors.Wrapf(e, "can not load hook [%v]", script.Name())
		}

		if e := toml.Unmarshal(contents, &h); e != nil {
			return errors.Wrapf(e, "can not unmarshal hook [%v]", script.Name())
		}

		t1 := false
		t2 := false

		if len(h.When) == 0 && when == "post" {
			t1 = true
		}

		if len(h.Operation) == 0 && opt == "install" {
			t2 = true
		}

		for _, v := range h.When {
			if v == when {
				t1 = true
				break
			}
		}

		for _, v := range h.Operation {
			if v == opt {
				t2 = true
				break
			}
		}

		if !t1 || !t2 {
			continue
		}

		for _, v := range h.Pattern {
			h.pattern = append(h.pattern, glob.MustCompile(v))
		}

		matches := &bytes.Buffer{}
		match := false

		// get matched files
	matched:
		for _, file := range files {
			for _, v := range h.pattern {
				if v.Match(file.Path) {
					match = true
					if h.NoArg {
						break matched
					}
					fmt.Fprintf(matches, "\"%s\" ", file.Path)
				}
			}
		}

		if match {
			command := fmt.Sprintf(`set -e
%s
_hookfunc() {
	%s
}
_hookfunc %s`,
				shenv.String(),
				h.Hook,
				matches.String())

			cmd := exec.Command("chroot", cfg.ROOT, "/bin/sh")
			cmd.Dir = "/"
			cmd.Stdin = strings.NewReader(command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if e := cmd.Run(); e != nil {
				return errors.Wrapf(e, "can not execute hook %v", script.Name())
			}
			log.Noticef("hook: executed hook %v", script.Name())
		}
	}

	return nil
}
