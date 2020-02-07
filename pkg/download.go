package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
)

func opt_download(name string, opts map[harg]bool) error {
	pkg, e := NewPkgfile(name, cfg.Ports)
	if e != nil {
		return errors.WithStack(e)
	}

	if e := pkg.Lock(); e != nil {
		return errors.WithStack(e)
	}
	defer pkg.Unlock()

	checksum := &bytes.Buffer{}

	for _, source := range pkg.Source {
		spath := filepath.Join(pkg.Dir, source["name"])
		log.Noticef("download[%v]: %+v", pkg.Name, source)

		switch source["protocol"] {
		case "git":
			var repo *git.Repository

			if exist(spath) {
				repo, e = git.PlainOpen(spath)
				if e != nil {
					os.RemoveAll(spath) // dirty dir, reclone
					goto notexisted
				}

				worktree, e := repo.Worktree()
				if e != nil {
					os.RemoveAll(spath) // dirty dir, reclone
					goto notexisted
				}

				ref, e := repo.Head()
				if e != nil {
					os.RemoveAll(spath) // dirty dir, reclone
					goto notexisted
				}

				if e := worktree.Reset(&git.ResetOptions{Commit: ref.Hash(), Mode: git.HardReset}); e != nil {
					os.RemoveAll(spath) // dirty dir, reclone
					goto notexisted
				}

				if e := worktree.Pull(&git.PullOptions{
					RemoteName: "origin",
					Progress:   os.Stdout,
				}); e != nil && e != git.NoErrAlreadyUpToDate {
					return errors.Wrapf(e, "can not pull [%v]", source["url"])
				}
			}

		notexisted:
			if !exist(spath) {
				repo, e = git.PlainClone(spath, false, &git.CloneOptions{
					URL:      source["url"],
					Progress: os.Stdout,
				})

				if e != nil {
					return errors.Wrapf(e, "can not clone [%v]", source["url"])
				}
			}

			if opts['c'] {
				if len(source["commit"]) != 0 {
					fmt.Fprintf(checksum, "%s %s\n", source["name"], source["commit"])
				} else {
					ref, e := repo.Head()
					if e != nil {
						return errors.Wrapf(e, "can not update checksum [%v]", source["url"])
					}

					fmt.Fprintf(checksum, "%s %s\n", source["name"], ref.Hash().String())
				}
			}
		default:
			switch source["protocol"] {
			case "http", "https":
				if exist(spath) {
					break
				}

				out, e := os.Create(spath)
				if e != nil {
					return errors.Wrapf(e, "can not create file [%v]", source["url"])
				}

				httpclient := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return nil
					},
				}

				resp, e := httpclient.Get(source["url"])
				if e != nil {
					os.Remove(spath)
					return errors.Wrapf(e, "can not send a http request [%v]", source["url"])
				}

				size := 0
				if len(resp.Header.Get("Content-Length")) != 0 {
					size, _ = strconv.Atoi(resp.Header.Get("Content-Length"))
				}

				counter := &progress{total: uint64(size)}
				_, e = io.Copy(out, io.TeeReader(resp.Body, counter))
				if e != nil {
					os.Remove(spath)
					return errors.Wrapf(e, "can not download source [%v]", source["url"])
				}

				fmt.Printf("\n")
				resp.Body.Close()
				out.Close()
			case "":
				if !exist(spath) {
					return errors.Errorf("except %s, but does not exist [%v]", source["name"], source["url"])
				}
			default:
				return errors.Errorf("unsupported protocol [%v, %v]", source["protocol"], source["url"])
			}

			if opts['c'] {
				sum := sha512.New()

				file, e := os.Open(spath)
				if e != nil {
					return errors.Wrapf(e, "can not open %v for checksum [%v]", spath, source["url"])
				}

				if _, e := io.Copy(sum, file); e != nil {
					return errors.Wrapf(e, "can not calculate checksum [%v]", source["url"])
				}
				file.Close()

				fmt.Fprintf(checksum, "%s %s\n", source["name"], hex.EncodeToString(sum.Sum(nil)))
			}
		}
	}

	if opts['c'] {
		if e := ioutil.WriteFile(pkg.Checksum, checksum.Bytes(), 0644); e != nil {
			return errors.Wrap(e, "can not update .checksum")
		}
		log.Noticef("download[%v]: updated .checksum", pkg.Name)
	}

	return nil
}
