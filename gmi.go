package gmi

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"sync"
)
import "golang.org/x/sync/errgroup"

type control struct {
	conn  *tls.Conn
	state Transition
	rules safemap
	g     *errgroup.Group
	ctx   context.Context
}
type safemap struct {
	sync.RWMutex
	m map[string]*rewriter
}
type rewriter struct {
	fn func(Node) string
	ch chan Node
}

func NewControl(ctx context.Context, isv Mask) *control {
	// encapsulate the key name from caller
	cx := context.WithValue(ctx, maskISVKey, isv)
	ctrl := &control{
		rules: safemap{m: make(map[string]*rewriter)},
		ctx:   cx,
	}

	ctrl.Attach(GmPlain, vanilla)
	ctrl.Attach(GmLink, rewriteLink)
	return ctrl
}

func (c *control) Dial(u *url.URL) (*bufio.Reader, error) {
	var (
		err            error
		status         int
		responseHeader string
	)
	if c.conn, err = dialTLS(c.ctx, u); err != nil {
		return nil, fmt.Errorf("Failed to connect: %w", err)
	}
	c.state = NetOpen
	// Send request (CR LF terminated)
	c.conn.Write([]byte(u.String() + "\r\n"))

	// Receive and parse response header
	reader := bufio.NewReader(c.conn)
	if responseHeader, err = reader.ReadString('\n'); err != nil {
		return c.dialError("Failed to read response %w", err)
	}
	// split on whitespace
	parts := strings.Fields(responseHeader)
	// status is two digits (but we only care about the leading digit)
	if status, err = strconv.Atoi(parts[0][0:1]); err != nil {
		return c.dialError("Failed to extract status %w", err)
	}

	switch status {
	case 1, 6:
		// No input, or client certs
		return c.dialError("Unsupported feature! - " + strconv.Itoa(status))

	case 2: // success
		// text/* content only
		meta := metaHeader(parts)
		if !strings.HasPrefix(meta, "text/") {
			return c.dialError("Not-implemented MIME support, " + meta)
		}
		return reader, nil

	case 3: // redirect
		// TODO ctx.Value(TLSRedirect) != 0
		meta := metaHeader(parts)
		if meta == "" {
			return c.dialError("REDIR meta header field error")
		}
		if lu, err := Format(meta, u.String()); err == nil {
			c.preRedirect()
			return c.Dial(lu)
		}
		return c.dialError("REDIR " + meta)
	case 4, 5:
		return c.dialError("ERROR: gemini status code 4 or 5")
	}

	return c.dialError("Exceptional status code did not match known values.")
}
func metaHeader(parts []string) string {
	if len(parts) < 2 {
		// missing meta field in the header
		////return c.dialError("Response header meta field")
		return ""
	}
	meta := parts[1]
	if len(meta) > 1024 {
		// cannot exceed 1024 bytes
		////return c.dialError("Response header size of meta field")
		return ""
	}
	return meta
}

// Gemtext op
const (
	GmPlain   = "catchall"
	GmLink    = "=>"
	GmHeading = "#"
	GmList    = "*"
	GmBlock   = ">"
	GmPrefmt  = "```"
)

func (c *control) Attach(op string, f func(Node) string) error {
	c.rules.Lock()
	defer c.rules.Unlock()

	// enum may reduce this error condition
	if op != GmLink && op != GmHeading && op != GmList &&
		op != GmBlock && op != GmPrefmt && op != GmPlain {
		return fmt.Errorf("Operation can only be =>, #, *, >, or ```")
	}

	/*
		// TODO implement AttachOrChain if supporting many rewriters per op
		if _, ok := c.rules.m[op]; ok {
			return fmt.Errorf("Rewriter already attached for %v", op)
		}*/

	var ch = make(chan Node)
	c.rules.m[op] = &rewriter{fn: f, ch: ch}

	return nil
}

func (c *control) Retrieve(r *bufio.Reader) (string, error) {
	c.rules.Lock()
	defer c.rules.Unlock()
	var (
		bld  strings.Builder
		buf  []byte
		err  error
		tree *Tree
		acc  = make(chan string)
	)
	// grab the entire gemini body since Parse() accepts the body as string
	if buf, err = ioutil.ReadAll(r); err != nil {
		return "", err
	}
	if tree, err = Parse(string(buf)); err != nil {
		return "", err
	}

	//TODO support cancellable
	grp, _ := errgroup.WithContext(c.ctx)
	go func() {
		// accumulate results
		for row := range acc {
			bld.WriteString(row)
		}
	}()
	// tree walk
	for _, no := range tree.Root.Nodes {
		switch no.Type() {
		case NodeLink:
			if run, ok := c.rules.m[GmLink]; ok {
				spawn(run.ch, run.fn, acc, grp)
				run.ch <- no
			}
		default:
			if run, ok := c.rules.m[GmPlain]; ok {
				spawn(run.ch, run.fn, acc, grp)
				run.ch <- no
			}
		}
	}

	// wait for grs to complete
	grp.Wait()
	// signal the for/range to end
	close(acc)

	return bld.String(), nil
}

// Disconnect and close gr channels
func (c *control) Close() {
	c.rules.Lock()
	defer c.rules.Unlock()

	if c.state != NetClose {
		//todo atomic set
		c.state = NetClose
		c.conn.Close()
	}
	for _, run := range c.rules.m {
		close(run.ch)
	}
}
func (c *control) preRedirect() {
	c.state = NetClose
	c.conn.Close()
}
func (c *control) dialError(ar string, e ...error) (*bufio.Reader, error) {
	// convenience to close connection, from dial errors
	c.preRedirect()
	if len(e) > 0 {
		return nil, fmt.Errorf(ar, e)
	}
	return nil, fmt.Errorf(ar)
}

// network state
type Transition uint8

const (
	NetNone Transition = iota
	NetOpen
	NetClose
)
