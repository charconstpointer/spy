package spy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/sync/errgroup"
)

type Watcher struct {
	c        http.Client
	hashes   map[string]*Result
	targets  []string
	E        chan *Result
	ticker   *time.Ticker
	hasher   func(string) (string, error)
	selector func(int, []string) int
	dmp      *diffmatchpatch.DiffMatchPatch
}
type WatcherOpts struct {
	Interval    time.Duration
	HTTPTimeout time.Duration
	Hasher      func(string) (string, error)
}
type Result struct {
	Hash string
	Diff string
}

func NewWatcher(opts WatcherOpts) *Watcher {
	return &Watcher{
		hashes: make(map[string]*Result),
		E:      make(chan *Result),
		c: http.Client{
			Timeout: opts.HTTPTimeout,
		},
		ticker: time.NewTicker(opts.Interval),
		hasher: opts.Hasher,
		selector: func(i int, s []string) int {
			i++
			i = i % len(s)
			return i
		},
		dmp: diffmatchpatch.New(),
	}
}

func (w *Watcher) Watch(ctx context.Context, targets ...string) error {
	for _, p := range targets {
		w.hashes[p] = nil
		w.targets = append(w.targets, p)
		fmt.Println("adding", p)
	}

	var g errgroup.Group
	g.Go(func() error {
		return w.watch(ctx)
	})

	return g.Wait()
}
func (w *Watcher) Write(ctx context.Context, wr ...io.Writer) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-w.E:
			for _, s := range wr {
				bw := bufio.NewWriter(s)
				fmt.Println("writing", s)
				fmt.Fprintln(bw, strings.Repeat("-", 80))
				fmt.Fprintf(bw, "hash\n%s\ndiff\n%s\n", ev.Hash, ev.Diff)
				fmt.Fprintln(bw, strings.Repeat("-", 80))
				bw.Flush()
			}
		}
	}
}
func (w *Watcher) watch(ctx context.Context) error {
	var pointer int
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.ticker.C:
			pointer = w.selector(pointer, w.targets)
			target := w.targets[pointer]
			fmt.Printf("[F.%d]:%s\n", pointer, target)
			result, err := w.fetch(ctx, target)
			if err != nil {
				return err
			}
			old := w.hashes[target]
			if old == nil {
				w.hashes[target] = result
				continue
			}
			if old != result {
				w.E <- result
			}
		}
	}
}

func (w *Watcher) fetch(ctx context.Context, url string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a request %w", err)
	}
	res, err := w.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %s %w", url, err)
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body %w", err)
	}
	hash, err := w.hasher(string(b))
	if err != nil {
		return nil, fmt.Errorf("could not hash %w", err)
	}
	var diff string
	if result := w.hashes[url]; result != nil {
		d := w.dmp.DiffMain(string(b), result.Diff, true)
		diff = w.dmp.DiffToDelta(d)
	} else {
		diff = string(b)
	}
	return &Result{
		Hash: hash,
		Diff: diff,
	}, nil
}
