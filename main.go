package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"strings"

	"github.com/charconstpointer/spy/spy"
)

var targets = flag.String("targets", "", "Comma separated list of targets")

func main() {
	flag.Parse()
	if *targets == "" {
		panic("No targets specified")
	}
	targets := strings.Split(*targets, ",")
	w := spy.NewWatcher(
		func(s string) (string, error) {
			return fmt.Sprintf("%x", md5.Sum([]byte(s))), nil
		},
	)
	ctx := context.Background()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-w.E:
				fmt.Println(ev)
			}
		}
	}()
	if err := w.Watch(ctx, targets...); err != nil {
		panic(err)
	}
}
