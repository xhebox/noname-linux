package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofrs/flock"
	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
)

type Pkgfile struct {
	Name     string
	Dir      string
	Version  string
	Desc     string
	Baks     []string
	Deps     []string
	Makedeps []string
	Ext      string
	Build    string
	NoStrip  bool
	KeepLa   bool
	Source   []map[string]string

	Builddir string
	Srcdir   string
	Pkgdir   string
	Tarball  string
	Checksum string

	env  *bytes.Buffer
	raw  *toml.Tree
	lock *flock.Flock
}

type byPos struct {
	raw  *toml.Tree
	keys []string
}

func (s byPos) Len() int {
	return len(s.keys)
}

func (s byPos) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s byPos) Less(i, j int) bool {
	x := s.raw.GetPosition(s.keys[i])
	y := s.raw.GetPosition(s.keys[j])

	if x.Line < y.Line {
		return true
	} else if x.Line == y.Line {
		return x.Col < y.Col
	} else {
		return false
	}
}

func NewPkgfile(name string, ports string) (*Pkgfile, error) {
	pkgfile := filepath.Join(ports, name, "Pkgfile")

	if exist(pkgfile) {
		pkg := &Pkgfile{Name: name, Dir: filepath.Join(ports, name)}

		var e error
		pkg.raw, e = toml.LoadFile(pkgfile)
		if e != nil {
			return nil, errors.Wrapf(e, "can not unmarshal pkgfile [%v]", name)
		}

		pkg.lock = flock.New(pkgfile)
		if r := pkg.raw.Get("build"); r != nil {
			pkg.Build = r.(string)
		} else {
			return nil, errors.Errorf("%v: build() can not be empty", name)
		}

		if r := pkg.raw.Get("desc"); r != nil {
			pkg.Desc = r.(string)
		}
		if r := pkg.raw.Get("ext"); r != nil {
			pkg.Ext = r.(string)
		}
		if r := pkg.raw.Get("keepla"); r != nil {
			pkg.KeepLa = r.(bool)
		}

		if r := pkg.raw.Get("baks"); r != nil {
			for _, v := range r.([]interface{}) {
				pkg.Baks = append(pkg.Baks, v.(string))
			}
		}

		if r := pkg.raw.Get("deps"); r != nil {
			for _, v := range r.([]interface{}) {
				pkg.Deps = append(pkg.Deps, v.(string))
			}
		}

		if r := pkg.raw.Get("makedeps"); r != nil {
			for _, v := range r.([]interface{}) {
				pkg.Makedeps = append(pkg.Makedeps, v.(string))
			}
		}

		{
			pkg.env = &bytes.Buffer{}

			fmt.Fprintf(pkg.env, "name=\"%s\"\n", pkg.Name)

			keys := byPos{pkg.raw, pkg.raw.Keys()}
			sort.Sort(keys)

			var srcs []*toml.Tree
			for _, v := range keys.keys {
				switch v {
				case "deps", "makedeps", "build", "ext":
				case "source":
					srcs = pkg.raw.Get("source").([]*toml.Tree)
				default:
					cmd := exec.Command("/bin/sh")
					cmd.Dir = "/"
					cmd.Stdin = strings.NewReader(
						fmt.Sprintf("%s\n printf \"%v\"", pkg.env.String(), pkg.raw.Get(v)))
					cmd.Stderr = os.Stderr

					pbytes, _ := cmd.Output()

					fmt.Fprintf(pkg.env, "%v=\"%v\"\n", v, string(pbytes))

					if v == "version" {
						pkg.Version = string(pbytes)
					}
				}
			}

			for _, src := range srcs {
				cmd := exec.Command("/bin/sh")
				cmd.Dir = "/"
				cmd.Stdin = strings.NewReader(
					fmt.Sprintf("%s\n printf \"%v\"", pkg.env.String(), src.Get("url")))
				cmd.Stderr = os.Stderr
				pbytes, _ := cmd.Output()
				url := string(pbytes)
				src.Set("url", url)

				// auto detect protocol & name
				if src.Get("protocol") == nil {
					if s := strings.SplitN(url, ":", 2); len(s) == 2 {
						src.Set("protocol", s[0])
					} else if s := strings.SplitN(url, "+", 2); len(s) == 2 {
						src.Set("protocol", s[0])
						src.Set("url", s[1])
					}
				}

				if src.Get("name") == nil {
					src.Set("name", path.Base(url))
				} else {
					cmd = exec.Command("/bin/sh")
					cmd.Dir = "/"
					cmd.Stdin = strings.NewReader(
						fmt.Sprintf("%s\nprintf \"%v\"", pkg.env.String(), src.Get("name")))
					cmd.Stderr = os.Stderr
					pbytes, _ = cmd.Output()
					src.Set("name", string(pbytes))
				}

				srcmap := map[string]string{}
				for _, v := range src.Keys() {
					srcmap[v] = fmt.Sprint(src.Get(v))
				}
				pkg.Source = append(pkg.Source, srcmap)
			}
		}

		fmt.Fprintf(pkg.env, "source=\"\n")
		for _, v := range pkg.Source {
			if len(v["name"]) != 0 {
				fmt.Fprintf(pkg.env, "%v\n", v["name"])
			} else {
				fmt.Fprintf(pkg.env, "%v\n", v["url"])
			}
		}
		fmt.Fprintf(pkg.env, "\"\n")

		pkg.Builddir = filepath.Join(cfg.BuildRoot, pkg.Name)
		pkg.Srcdir = filepath.Join(cfg.BuildRoot, pkg.Name, "src")
		pkg.Pkgdir = filepath.Join(cfg.BuildRoot, pkg.Name, "pkg")
		fmt.Fprintf(pkg.env, "srcdir=\"%v\"\n", pkg.Srcdir)
		fmt.Fprintf(pkg.env, "pkgdir=\"%v\"\n", pkg.Pkgdir)

		pkg.Tarball = filepath.Join(pkg.Dir, pkg.Name+"#"+pkg.Version+".tgz")
		pkg.Checksum = filepath.Join(pkg.Dir, ".checksum")

		return pkg, nil
	}

	return nil, errors.Errorf("%s not found", name)
}

func (c *Pkgfile) Lock() error {
	ok, e := c.lock.TryLock()

	if !ok {
		return errors.Wrapf(e, "one process one time, but can not flock %v", c.Name)
	}

	return nil
}

func (c *Pkgfile) Unlock() error {
	if e := c.lock.Close(); e != nil {
		return errors.WithStack(e)
	}

	return nil
}

func (c *Pkgfile) MkSrcDir() error {
	if e := os.RemoveAll(c.Srcdir); e != nil {
		return errors.Wrapf(e, "can not remove src [%v]", c.Name)
	}

	if e := os.MkdirAll(c.Srcdir, 0755); e != nil {
		return errors.Wrapf(e, "can not mkdir src [%v]", c.Name)
	}

	return nil
}

func (c *Pkgfile) MkPkgDir() error {
	if e := os.RemoveAll(c.Pkgdir); e != nil {
		return errors.Wrapf(e, "can not remove pkg [%v]", c.Name)
	}

	if e := os.MkdirAll(c.Pkgdir, 0755); e != nil {
		return errors.Wrapf(e, "can not mkdir pkg [%v]", c.Name)
	}

	return nil
}

func (c *Pkgfile) RmDirs() error {
	if e := os.RemoveAll(c.Builddir); e != nil {
		return errors.Wrapf(e, "can not remove src [%v]", c.Name)
	}

	return nil
}

func (c *Pkgfile) PackageTarball() error {
	if e := os.Chdir(c.Pkgdir); e != nil {
		return errors.WithStack(e)
	}

	tarballf, e := os.OpenFile(c.Tarball, os.O_RDWR|os.O_CREATE, 0644)
	if e != nil {
		return errors.Wrapf(e, "can not create %v [%v]", c.Tarball, c.Name)
	}
	defer tarballf.Close()

	gzwt := gzip.NewWriter(tarballf)
	defer gzwt.Close()

	tarwt := tar.NewWriter(gzwt)
	defer tarwt.Close()

	if e := filepath.Walk(".", func(spath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "can not walk files to package [%v]", c.Name)
		}

		if spath == "." {
			return nil
		}

		mode := info.Mode()
		if mode&os.ModeType != 0 && mode&os.ModeDir == 0 && mode&os.ModeSymlink == 0 {
			return errors.Errorf("unsupported file type to package [%v]", c.Name)
		}

		hdr, e := tar.FileInfoHeader(info, "")
		if e != nil {
			return errors.Wrapf(e, "can not generate tar hdr from info %+v [%v]", info, c.Name)
		}

		hdr.Name = filepath.Join(spath)
		if hdr.Linkname == "" && mode&os.ModeSymlink != 0 {
			hdr.Linkname, e = os.Readlink(spath)
			if e != nil {
				return errors.Wrapf(e, "can not correct link name of header [%v]", c.Name)
			}
		}

		isFile := !info.IsDir() && (info.Mode()&os.ModeSymlink == 0)
		var fp *os.File

		if isFile {
			fp, e = os.Open(spath)
			if e != nil {
				return errors.Wrapf(e, "can not open %v for packaging [%v]", spath, c.Name)
			}
			defer fp.Close()

			hdr.PAXRecords = make(map[string]string)

			attrs, e := xattr.FList(fp)
			if e != nil {
				return errors.Wrapf(e, "can not list attrs [%v]", c.Name)
			}

			for _, attr := range attrs {
				attrval, e := xattr.FGet(fp, attr)
				if e != nil {
					return errors.Wrapf(e, "can not get attr [%v]", c.Name)
				}

				hdr.PAXRecords["SCHILY.xattr."+attr] = string(attrval)
			}
		}

		if e := tarwt.WriteHeader(hdr); e != nil {
			return errors.Wrapf(e, "can not write header to disk [%v]", c.Name)
		}

		if isFile {
			if _, e := io.Copy(tarwt, fp); e != nil {
				return errors.Wrapf(e, "can not write %v to tarball [%v]", spath, c.Name)
			}
		}

		return nil
	}); e != nil {
		return errors.WithStack(e)
	}

	return nil
}

func (c *Pkgfile) ShEnv() string {
	return c.env.String()
}

func (c *Pkgfile) Footprint() ([]Fileinfo, error) {
	ret := []Fileinfo{}

	fd, e := os.Open(c.Tarball)
	if e != nil {
		return ret, errors.Wrapf(e, "can not open tarball for footprint [%v]", c.Tarball)
	}
	defer fd.Close()

	gzrd, e := gzip.NewReader(fd)
	if e != nil {
		return ret, errors.Wrapf(e, "can not ungzip tarball for footprint [%v]", c.Tarball)
	}
	defer gzrd.Close()

	tard := tar.NewReader(gzrd)
	for {
		hdr, e := tard.Next()
		if e == io.EOF {
			break
		} else if e != nil {
			return ret, errors.Wrapf(e, "can not untar tarball for footprint [%v]", c.Tarball)
		}

		ret = append(ret, Fileinfo{
			Path: hdr.Name,
			Uid:  hdr.Uid,
			Gid:  hdr.Gid,
			Mode: hdr.Mode,
			Size: hdr.Size,
			Type: hdr.Typeflag,
		})
	}

	return ret, nil
}
