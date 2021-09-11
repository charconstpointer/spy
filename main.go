package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charconstpointer/spy/spy"
)

var (
	targets = flag.String("targets", "", "Comma separated list of targets")
)

func main() {
	flag.Parse()
	if *targets == "" {
		panic("No targets specified")
	}
	targets := strings.Split(*targets, ",")

	opts := spy.WatcherOpts{
		Hasher: func(s string) (string, error) {
			return fmt.Sprintf("%x", md5.Sum([]byte(s))), nil
		},
		HTTPTimeout: time.Duration(5) * time.Second,
		Interval:    time.Duration(1) * time.Second,
	}
	w := spy.NewWatcher(opts)
	ctx := context.Background()
	file, err := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	go w.Write(ctx, file)
	if err := w.Watch(ctx, targets...); err != nil {
		panic(err)
	}
}
