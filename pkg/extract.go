package main

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	dircopy "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func opt_extract(name string, opts map[harg]bool) error {
	pkg, e := NewPkgfile(name, cfg.Ports)
	if e != nil {
		return errors.WithStack(e)
	}

	if e := pkg.Lock(); e != nil {
		return errors.Wrapf(e, "one process one time, but can not flock %v", name)
	}
	defer pkg.Unlock()

	if e := pkg.MkSrcDir(); e != nil {
		return errors.WithStack(e)
	}

	var checksum *bufio.Scanner
	if !opts['m'] {
		f, e := os.Open(pkg.Checksum)
		if e != nil {
			return errors.Wrapf(e, "can not read .checksum for %v", pkg.Name)
		}
		defer f.Close()

		checksum = bufio.NewScanner(f)
	}

	for _, source := range pkg.Source {
		spath := filepath.Join(pkg.Dir, source["name"])
		log.Noticef("extract[%v]: %+v", pkg.Name, source)

		if !exist(spath) {
			return errors.Errorf("except %s, but does not exist [%v]", source["name"], source["url"])
		}

		checksum.Scan()
		switch source["protocol"] {
		case "git":
			repo, e := git.PlainOpen(spath)
			if e != nil {
				return errors.Wrapf(e, "can not open [%v]", source["url"])
			}

			if len(source["commit"]) != 0 {
				worktree, e := repo.Worktree()
				if e != nil {
					return errors.Wrapf(e, "can not get worktree [%v]", source["url"])
				}

				if e := worktree.Reset(&git.ResetOptions{Commit: plumbing.NewHash(source["commit"]), Mode: git.HardReset}); e != nil {
					return errors.Wrapf(e, "can not reset to head [%v]", source["url"])
				}
			}

			if !opts['m'] {
				ref, e := repo.Head()
				if e != nil {
					return errors.Wrapf(e, "can not get head hash [%v]", source["url"])
				}

				sum := fmt.Sprintf("%s %s", source["name"], ref.Hash().String())
				if sum != checksum.Text() {
					return errors.Errorf("commit hash is different[%v|%v]", sum, checksum.Text())
				}
			}

			if e := dircopy.Copy(spath, filepath.Join(pkg.Srcdir, source["name"])); e != nil {
				return errors.Wrapf(e, "can not copy [%v]", source["url"])
			}
		default:
			switch source["protocol"] {
			case "http", "https", "":
			default:
				return errors.Errorf("unsupported protocol [%v, %v]", source["protocol"], source["url"])
			}

			if !opts['m'] {
				hash := sha512.New()

				file, e := os.Open(spath)
				if e != nil {
					return errors.Wrapf(e, "can not open [%v]", source["url"])
				}

				if _, e := io.Copy(hash, file); e != nil {
					return errors.Wrapf(e, "can not calculate checksum for %v", name)
				}

				sum := fmt.Sprintf("%s %s", source["name"], hex.EncodeToString(hash.Sum(nil)))
				if sum != checksum.Text() {
					return errors.Errorf("hash is different[%v %v]", sum, checksum.Text())
				}

				file.Close()
			}

			command := ""
			switch {
			case strings.HasSuffix(spath, ".tgz"), strings.HasSuffix(spath, ".tar.gz"), strings.HasSuffix(spath, ".txz"), strings.HasSuffix(spath, ".tar.xz"), strings.HasSuffix(spath, ".tbz2"), strings.HasSuffix(spath, ".tar.bz2"), strings.HasSuffix(spath, ".tlz4"), strings.HasSuffix(spath, ".tar.lz4"), strings.HasSuffix(spath, ".tsz"), strings.HasSuffix(spath, ".tar.sz"), strings.HasSuffix(spath, ".tar"),
				strings.HasSuffix(spath, ".tlz"), strings.HasSuffix(spath, ".tar.lz"),
				strings.HasSuffix(spath, ".tar.zst"):
				if exist("/bin/bsdtar") || exist("/usr/bin/bsdtar") {
					command = "bsdtar xpf " + spath
				} else if exist("/bin/tar") || exist("/usr/bin/tar") {
					command = "tar xpf " + spath
				}
			case strings.HasSuffix(spath, ".zip"):
				if exist("/bin/unzip") || exist("/usr/bin/unzip") {
					command = "unzip -qXo " + spath
				} else if exist("/bin/7z") || exist("/usr/bin/7z") {
					command = "7z x " + spath
				}
			case strings.HasSuffix(spath, ".rar"):
				if exist("/bin/unrar") || exist("/usr/bin/unrar") {
					command = "unrar e " + spath
				} else if exist("/bin/7z") || exist("/usr/bin/7z") {
					command = "7z x " + spath
				}
			case strings.HasSuffix(spath, ".gz"):
				if exist("/bin/gzip") || exist("/usr/bin/gzip") {
					command = "gzip -dkc " + spath + " > " + filepath.Join(pkg.Srcdir, strings.TrimSuffix(source["name"], ".gz"))
				} else if exist("/bin/pigz") || exist("/usr/bin/pigz") {
					command = "pigz -dkc " + spath + " > " + filepath.Join(pkg.Srcdir, strings.TrimSuffix(source["name"], ".gz"))
				}
			case strings.HasSuffix(spath, ".xz"):
				if exist("/bin/unxz") || exist("/usr/bin/unxz") {
					command = "unxz -dkc " + spath + " > " + filepath.Join(pkg.Srcdir, strings.TrimSuffix(source["name"], ".xz"))
				}
			case strings.HasSuffix(spath, ".bz2"):
				if exist("/bin/bzip2") || exist("/usr/bin/bzip2") {
					command = "bzip2 -dkc " + spath + " > " + filepath.Join(pkg.Srcdir, strings.TrimSuffix(source["name"], ".bz2"))
				} else if exist("/bin/pbzip2") || exist("/usr/bin/pbzip2") {
					command = "pbzip2 -dkc " + spath + " > " + filepath.Join(pkg.Srcdir, strings.TrimSuffix(source["name"], ".bz2"))
				}
			default:
				if e := os.Link(spath, filepath.Join(pkg.Srcdir, source["name"])); e != nil {
					return errors.Wrapf(e, "can not hardlink [%v]", source["url"])
				}

				continue
			}

			if len(command) == 0 {
				return errors.Errorf("no tools to extract [%v]", source["url"])
			}

			cmd := exec.Command("/bin/sh")
			cmd.Dir = pkg.Srcdir
			cmd.Stdin = strings.NewReader(command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if e := cmd.Run(); e != nil {
				return errors.Wrapf(e, "can not extract [%v]", source["url"])
			}
		}
	}

	if len(pkg.Ext) != 0 {
		cmd := exec.Command("/bin/sh")
		cmd.Dir = pkg.Srcdir
		cmd.Stdin = strings.NewReader(fmt.Sprintf(`%s
%s
%s`,
			shenv.String(),
			pkg.ShEnv(),
			pkg.Ext))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Noticef("build[%v]: ext()", pkg.Name)
		fmt.Fprintf(os.Stderr, "\033[1;0m\b")
		if e := cmd.Run(); e != nil {
			return errors.Wrapf(e, "can not execute ext() [%v]", pkg.Name)
		}
	}

	return nil
}
