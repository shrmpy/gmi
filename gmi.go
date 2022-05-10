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

	"golang.org/x/sync/errgroup"
)

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

func NewControl(ctx context.Context) *control {
	ctrl := &control{
		rules: safemap{m: make(map[string]*rewriter)},
		ctx:   ctx,
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
	////if c.conn, err = tls.Dial("tcp", u.Host, nil); err != nil {
	if c.conn, err = dialTLS(u); err != nil {
		return nil, fmt.Errorf("Failed to connect: %v", err)
	}
	c.state = NetOpen
	// Send request (CR LF terminated)
	c.conn.Write([]byte(u.String() + "\r\n"))

	// Receive and parse response header
	reader := bufio.NewReader(c.conn)
	if responseHeader, err = reader.ReadString('\n'); err != nil {
		return c.dialError("Failed to read response %v", err.Error())
	}
	// split on whitespace
	parts := strings.Fields(responseHeader)
	// status is two digits (but we only care about the leading digit)
	if status, err = strconv.Atoi(parts[0][0:1]); err != nil {
		return c.dialError("Failed to extract status %v", err.Error())
	}
	meta := parts[1]
	if len(meta) > 1024 {
		// cannot exceed 1024 bytes
		return c.dialError("Response header size of meta field")
	}

	switch status {
	case 1, 6:
		// No input, or client certs
		return c.dialError("Unsupported feature! - %s", strconv.Itoa(status))

	case 2: // success
		// text/* content only
		if !strings.HasPrefix(meta, "text/") {
			return c.dialError("Not-implemented MIME support %s", meta)
		}
		return reader, nil

	case 3: // redirect
		// TODO use config setting (enable-follow as default)
		if lu, err := Format(meta, u.String()); err == nil {
			c.preRedirect()
			return c.Dial(lu)
		}
		return c.dialError("REDIR %s", meta)
	case 4, 5:
		return c.dialError("ERROR: %s", meta)
	}

	return c.dialError("Exceptional status code did not match known values.")
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
func (c *control) dialError(ar ...string) (*bufio.Reader, error) {
	// convenience to close connection, from dial errors
	c.preRedirect()
	if len(ar) > 1 {
		return nil, fmt.Errorf(ar[0], ar[1:])
	}
	return nil, fmt.Errorf(ar[0])
}

// network state
type Transition uint8

const (
	NetNone Transition = iota
	NetOpen
	NetClose
)
