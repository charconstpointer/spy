package spy

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

type Watcher struct {
	c       http.Client
	hashes  map[string]string
	targets []string
	E       chan string
	ticker  *time.Ticker
	hasher  func(string) (string, error)
}

func NewWatcher(hasher func(string) (string, error)) *Watcher {
	return &Watcher{
		hashes: make(map[string]string),
		E:      make(chan string),
		c: http.Client{
			Timeout: time.Second * 5,
		},
		ticker: time.NewTicker(time.Second * 1),
		hasher: hasher,
	}
}

func (w *Watcher) Watch(ctx context.Context, targets ...string) error {
	for _, p := range targets {
		w.hashes[p] = ""
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
			pointer++
			pointer = pointer % len(w.targets)
			target := w.targets[pointer]
			fmt.Printf("[F.%d]:%s\n", pointer, target)
			hash, err := w.fetch(ctx, target)
			if err != nil {
				return err
			}
			old := w.hashes[target]
			if old == "" {
				w.hashes[target] = hash
				continue
			}
			if old != hash {
				w.E <- hash
			}
		}
	}
}

func (w *Watcher) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("could not create a request %w", err)
	}
	res, err := w.c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not fetch %s %w", url, err)
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read body %w", err)
	}
	hash, err := w.hasher(string(b))
	if err != nil {
		return "", fmt.Errorf("could not hash %w", err)
	}
	return hash, nil
}
