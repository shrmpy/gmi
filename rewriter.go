package gmi

import (
	"fmt"

	"golang.org/x/sync/errgroup"
)

// skeleton rule to demonstrate GEMtext link lines.
func rewriteLink(n Node) string {
	return fmt.Sprintf("\n[+] %s", n.String())
}

// A default rule for GEMtext plain text lines.
func vanilla(n Node) string {
	return fmt.Sprintf("\n%s", n.String())
}

// use a wrapper to handle (enforce) the channels to/from the func
func spawn(ch <-chan Node, f func(Node) string, out chan<- string, g *errgroup.Group) {
	g.Go(func() error {
		var node = <-ch
		out <- f(node)
		return nil
	})
}
