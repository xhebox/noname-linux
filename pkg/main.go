package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	logger "github.com/apsdehal/go-logger"
	toml "github.com/pelletier/go-toml"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
	"github.com/timtadh/getopt"
)

// redefine arg to print char instead of number
type harg byte

func (c harg) String() string {
	return fmt.Sprintf("%c", c)
}

type config struct {
	HTTP_PROXY  string   `toml:"http_proxy"`
	HTTPS_PROXY string   `toml:"https_proxy"`
	ARCH        string   `toml:"arch"`
	TARGET      string   `toml:"target"`
	OTARGETS    []string `toml:"other_targets"`
	ROOT        string   `toml:"root"`
	BuildRoot   string   `toml:"buildroot"`
	Database    string   `toml:"database"`
	Hooks       string   `toml:"hooks"`
	Color       bool     `toml:"color"`
	Ports       string   `toml:"ports"`
	CFLAGS      string   `toml:"CFLAGS"`
	CXXFLAGS    string   `toml:"CXXFLAGS"`
	LDFLAGS     string   `toml:"LDFLAGS"`
	Makeflags   int      `toml:"makeflags"`
	CC          string   `toml:"CC"`
	CXX         string   `toml:"CXX"`
	LD          string   `toml:"LD"`
	Ccache      bool     `toml:"ccache"`
}

var (
	log, _ = logger.New(os.Stdout)
	cfg    = config{}
	db     *diskv.Diskv
	shenv  = &bytes.Buffer{}
)

func main() {
	log.SetFormat("%{file}:%{line} %{message}")
	log.SetLogLevel(logger.NoticeLevel)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Fatalf("captured signal %v\n", sig)
		}
	}()

	// subaction [z] is reversed for internal use
	pkgs, args, e := getopt.GetOpt(os.Args[1:], "EUMQSRhabcdefiklmoprstuxyvz", []string{"config=", "debug", "root="})
	if e != nil {
		log.Fatalf(errors.Wrapf(e, "can not parse arguments").Error())
	}

	cfgfile := "/etc/pkg/config"
	root := ""
	mact := harg('h')
	sact := make(map[harg]bool)
	for k := range args {
		arg := args[k]

		switch act := arg.Opt(); act {
		case "-U", "-M", "-Q", "-S", "-R", "-E", "-h":
			mact = harg(act[1])
		case "-a", "-b", "-c", "-d", "-f", "-i", "-k", "-l", "-m", "-o", "-p", "-r", "-s", "-t", "-u", "-x", "-y", "-z":
			sact[harg(act[1])] = true
		case "--config":
			cfgfile = arg.Arg()
		case "--root":
			root = arg.Arg()
		case "--debug", "-v":
			log.SetLogLevel(logger.DebugLevel)
		}
	}

	cfgbytes, e := ioutil.ReadFile(cfgfile)
	if e != nil {
		log.Fatalf(errors.Wrapf(e, "can not load the config file").Error())
	}

	if e := toml.Unmarshal(cfgbytes, &cfg); e != nil {
		log.Fatalf(errors.Wrapf(e, "can not parse the config file").Error())
	}

	if cfg.Makeflags == 0 {
		cfg.Makeflags = runtime.NumCPU()
	}

	if e := os.Chdir(os.TempDir()); e != nil {
		log.Fatalf(errors.Wrapf(e, "can not enter an temp workdir").Error())
	}

	if len(root) != 0 {
		cfg.ROOT = root
	} else {
		cfg.ROOT, e = filepath.Abs(cfg.ROOT)
		if e != nil {
			log.Fatalf(errors.Wrapf(e, "can not make [root] an absolute path").Error())
		}
	}

	cfg.Hooks, e = filepath.Abs(filepath.Join(cfg.ROOT, cfg.Hooks))
	if e != nil {
		log.Fatalf(errors.Wrapf(e, "can not make [hooks] an absolute path").Error())
	}

	cfg.Database, e = filepath.Abs(filepath.Join(cfg.ROOT, cfg.Database))
	if e != nil {
		log.Fatalf(errors.Wrapf(e, "can not make [database] an absolute path").Error())
	}

	log.Debugf("config: %+v, mact: %c, sact: %+v, args: %v\n", cfg, mact, sact, pkgs)

	db = diskv.New(diskv.Options{
		BasePath: cfg.Database,
		Transform: func(key string) []string {
			return []string{}
		},
		CacheSizeMax: 1024 * 1024,
		Compression:  diskv.NewZlibCompression(),
	})

	os.Setenv("HTTP_PROXY", cfg.HTTP_PROXY)
	os.Setenv("HTTPS_PROXY", cfg.HTTPS_PROXY)

	fmt.Fprintln(shenv, "set -e")
	fmt.Fprintln(shenv, "export QMAKE_CFLAGS_ISYSTEM=")
	fmt.Fprintf(shenv, "ARCH=%s\n", cfg.ARCH)
	fmt.Fprintf(shenv, "TARGET=%s\n", cfg.TARGET)
	fmt.Fprintf(shenv, "OTARGETS=\"")
	for i := range cfg.OTARGETS {
		if i != 0 {
			fmt.Fprintf(shenv, " ")
		}
		fmt.Fprintf(shenv, "%s", cfg.OTARGETS[i])
	}
	fmt.Fprintf(shenv, "\"\n")
	fmt.Fprintf(shenv, "ROOT=%s\n", cfg.ROOT)
	fmt.Fprintf(shenv, "DB=%s\n", cfg.Database)
	fmt.Fprintf(shenv, "HOOKS=%s\n", cfg.Hooks)
	fmt.Fprintf(shenv, "export HTTPS_PROXY=%s\n", cfg.HTTPS_PROXY)
	fmt.Fprintf(shenv, "export HTTP_PROXY=%s\n", cfg.HTTP_PROXY)
	if len(cfg.CC) != 0 {
		fmt.Fprintf(shenv, "export CC=\"%s\"\n", cfg.CC)
	}
	if len(cfg.CXX) != 0 {
		fmt.Fprintf(shenv, "export CXX=\"%s\"\n", cfg.CXX)
	}
	if len(cfg.LD) != 0 {
		fmt.Fprintf(shenv, "export LD=\"%s\"\n", cfg.LD)
	}
	fmt.Fprintf(shenv, "export CFLAGS=\"%s\"\n", cfg.CFLAGS)
	fmt.Fprintf(shenv, "export CXXFLAGS=\"%s\"\n", cfg.CXXFLAGS)
	fmt.Fprintf(shenv, "export LDFLAGS=\"%s\"\n", cfg.LDFLAGS)
	fmt.Fprintf(shenv, "export MAKEFLAGS=\"-j%d\"\n", cfg.Makeflags)
	if cfg.Ccache {
		fmt.Fprintln(shenv, "export PATH=\"/lib/ccache:$PATH\"")
	}

	switch mact {
	case 'E':
		for _, pkg := range pkgs {
			if e := opt_extract(pkg, sact); e != nil {
				log.Fatalf("%+v\n", e)
			}
		}
	case 'U':
		for _, pkg := range pkgs {
			if e := opt_download(pkg, sact); e != nil {
				log.Fatalf("%+v\n", e)
			}
		}
	case 'M':
		for _, pkg := range pkgs {
			if e := opt_build(pkg, sact); e != nil {
				log.Fatalf("%+v\n", e)
			}
		}
	case 'S':
		if e := opt_install(pkgs, sact); e != nil {
			log.Fatalf("%+v\n", e)
		}
	case 'R':
		if e := opt_remove(pkgs, sact); e != nil {
			log.Fatalf("%+v\n", e)
		}
	case 'Q':
		if sact['a'] {
			if e := opt_query("", sact); e != nil {
				log.Fatalf("%+v\n", e)
			}
		} else if sact['t'] {
			pattern := "*"
			if len(pkgs) != 0 {
				pattern = strings.Join(pkgs, " ")
			}

			if e := opt_rmnoneed(pattern); e != nil {
				log.Fatalf("%+v\n", e)
			}
		} else {
			for _, pkg := range pkgs {
				if e := opt_query(pkg, sact); e != nil {
					log.Fatalf("%+v\n", e)
				}
			}
		}
	default:
		log.Notice("pkg [actions(subactions)] ([common]) (args)")
		log.Notice("")
		log.Notice("[common] [function]")
		log.Notice("-h print this usage")
		log.Notice("--debug(-v) enable verbose output")
		log.Notice("--config specify the config file")
		log.Notice("--root specify the root")
		log.Notice("")
		log.Notice("[action] [function]")
		log.Notice("-U fetch packages")
		log.Notice("\t(c) update .checksum")
		log.Notice("-E extract packages")
		log.Notice("\t(m) ignore checksum through extraction")
		log.Notice("-M fetch, extract and build packages")
		log.Notice("\t(d) no makedeps")
		log.Notice("\t(k) keep src & pkg")
		log.Notice("\t(x) assume extraction has done")
		log.Notice("-S install packages")
		log.Notice("\t(d) no deps")
		log.Notice("\t(r) upgrade installed package")
		log.Notice("\t(z) not to mark it as a user-installed package")
		log.Notice("-R remove packages")
		log.Notice("\t(d) only remove specific packages")
		log.Notice("-Q query system or packages or files, [i] is the default subaction if no provided one")
		log.Notice("\t(a) list all installed packages")
		log.Notice("\t(p) list all packages match the wildcard pattern")
		log.Notice("\t(t) remove unneeded and non-user-installed packages that mached the wildcard pattern, default pattern to *")
		log.Notice("\t(l) print the footprint")
		log.Notice("\t(i) print info of the package")
		log.Notice("\t(o) look for the owner of the file/dir")
		log.Notice("\t(d) look for packages depends on")
	}
}
