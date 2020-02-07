package main

import (
	toml "github.com/pelletier/go-toml"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
)

type Fileinfo struct {
	Path string
	Uid  int
	Gid  int
	Mode int64
	Size int64
	Type byte
}

type Pkgdb struct {
	Name      string
	Version   string     `toml:"version"`
	Deps      []string   `toml:"deps"`
	Desc      string     `toml:"desc"`
	Baks      []string   `toml:"bak"`
	Footprint []Fileinfo `toml:"footprint"`
	User      bool       `toml:"user"`
}

func NewPkgdb(name string, db *diskv.Diskv) (*Pkgdb, error) {
	pbytes, e := db.Read(name)
	if e != nil {
		return nil, errors.Wrapf(e, "can not load pkgdb [%v]", name)
	}

	c := &Pkgdb{Name: name}
	if e := toml.Unmarshal(pbytes, c); e != nil {
		return nil, errors.Wrapf(e, "can not unmarshal pkgdb [%v]", name)
	}

	return c, nil
}

func (c *Pkgdb) Store(db *diskv.Diskv) error {
	pbytes, e := toml.Marshal(c)
	if e != nil {
		return errors.Wrapf(e, "can not unmarshal pkgdb [%v]", c.Name)
	}

	if e := db.Write(c.Name, pbytes); e != nil {
		return errors.Wrapf(e, "can not store pkgdb [%v]", c.Name)
	}

	return nil
}

func Erase(name string, db *diskv.Diskv) error {
	if e := db.Erase(name); e != nil {
		return errors.Wrapf(e, "can not erase pkgdb [%v]", name)
	}

	return nil
}
