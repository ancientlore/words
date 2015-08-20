package main

import (
	"fmt"
	"golang.org/x/net/context"
	"regexp"
	"sort"
	"strings"
	"time"
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
	tck := time.NewTicker(1 * time.Minute)

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
	for i := len(sl) - 1; i >= 0; i-- {
		fmt.Println(sl[i])
	}
}

func main() {
	ctx := context.Background()
	rch := reader(ctx)
	accumulator(ctx, rch)
}
