package main

import (
	"fmt"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
)

// used when downloading
type progress struct {
	t     time.Time
	speed uint64
	prev  uint64
	now   uint64
	total uint64
}

func (c *progress) Write(p []byte) (int, error) {
	n := len(p)
	c.now += uint64(n)

	now := time.Now()
	duration := now.Sub(c.t)
	if duration.Seconds() > 1 {
		c.speed = uint64(float64(c.now-c.prev) / duration.Seconds())
		c.prev = c.now
		c.t = now
	}
	fmt.Printf("\r%s", strings.Repeat(" ", 60))
	fmt.Printf("\r[%s/%s] ...... %s/s",
		humanize.IBytes(c.now),
		humanize.IBytes(c.total),
		humanize.IBytes(c.speed))

	return n, nil
}
