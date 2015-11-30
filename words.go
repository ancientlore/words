package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	top      = flag.Int("top", 0, "Only print top entries")
	interval = flag.Duration("interval", time.Minute, "Print interval")
)

func reader(ctx context.Context) chan string {
	done := ctx.Done()
	out := make(chan string)
	re := regexp.MustCompile("[^\\p{L}\\p{Ll}\\p{N}#@]+")

	f := func() {
		defer close(out)
		var s string
		for n, err := fmt.Scan(&s); err == nil && n > 0; n, err = fmt.Scan(&s) {
			s = re.ReplaceAllString(s, " ")
			st := strings.Split(s, " ")
			for _, v := range st {
				if v != "" && !strings.HasPrefix(v, "@") {
					select {
					case out <- strings.ToLower(v):
					case <-done:
						return
					}
				}
			}
		}
	}
	go f()
	return out
}

func accumulator(ctx context.Context, ch chan string) {
	done := ctx.Done()
	tck := time.NewTicker(*interval)

	m := make(map[string]int)
	for {
		select {
		case s, op := <-ch:
			if !op {
				print(m)
				return
			}
			if x, ok := m[s]; ok {
				m[s] = x + 1
			} else {
				m[s] = 1
			}
		case <-tck.C:
			print(m)
		case <-done:
			print(m)
			return
		}
	}
}

func print(m map[string]int) {
	var sl []string
	for s, x := range m {
		sl = append(sl, fmt.Sprintf("%8d %s", x, s))
	}
	sort.Strings(sl)
	fmt.Println()
	min := 0
	if *top > 0 {
		min = len(sl) - *top
		if min < 0 {
			min = 0
		}
	}
	for i := len(sl) - 1; i >= min; i-- {
		fmt.Println(sl[i])
	}
}

func main() {
	flag.Parse()

	// get context
	ctx, cancel := context.WithCancel(context.Background())

	// start reader
	rch := reader(ctx)

	// handle kill signals
	go func() {
		// Set up channel on which to send signal notifications.
		// We must use a buffered channel or risk missing the signal
		// if we're not ready to receive when the signal is sent.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		// Block until a signal is received.
		<-c
		cancel()
	}()

	// accumulate results
	accumulator(ctx, rch)
}
