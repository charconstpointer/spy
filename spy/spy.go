package spy

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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
}

func NewWatcher(hasher func(string) (string, error)) *Watcher {
	return &Watcher{
		hashes: make(map[string]*Result),
		E:      make(chan *Result),
		c: http.Client{
			Timeout: time.Second * 5,
		},
		ticker: time.NewTicker(time.Millisecond * 10000),
		hasher: hasher,
		selector: func(i int, s []string) int {
			i++
			i = i % len(s)
			return i
		},
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

type Result struct {
	Hash string
	Diff string
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
		diff = diff2(string(b), result.Diff)
	} else {
		diff = string(b)
	}
	return &Result{
		Hash: hash,
		Diff: diff,
	}, nil
}

func diff2(a, b string) string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range strings.Split(a, "\n") {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range strings.Split(b, "\n") {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return strings.Join(diff, "\n")
}
