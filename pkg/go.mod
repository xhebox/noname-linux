module github.com/xhebox/noname-linux/pkg

replace github.com/xhebox/noname-linux/pkg => ./

go 1.13

require (
	github.com/apsdehal/go-logger v0.0.0-20190515212710-b0d6ccfee0e6
	github.com/dustin/go-humanize v1.0.0
	github.com/go-git/go-git/v5 v5.1.0
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/flock v0.7.1
	github.com/otiai10/copy v1.0.2
	github.com/pelletier/go-toml v1.6.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.8.1
	github.com/pkg/xattr v0.4.1
	github.com/timtadh/getopt v1.0.1
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527
	gopkg.in/src-d/go-git.v4 v4.13.1
	v.io/x/lib v0.1.4
)
