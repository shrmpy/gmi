package gmi

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	CA_ROOTS         = "/etc/ssl/certs/ca-certificates.crt"
	CONNECTION_OPEN  = 100
	CONNECTION_CLOSE = 110
)

type control struct {
	conn  *tls.Conn
	state int
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

// format the URL for Gemini scheme
func Format(raw string, referer string) (*url.URL, error) {
	//TODO make unit tests to prove we follow
	//     https://gemini.circumlunar.space/docs/specification.gmi
	var err error
	// standard port is 1965
	var rfr = &url.URL{Scheme: "gemini", Host: ":1965"}
	if strings.HasPrefix(referer, "gemini://") {
		if rfr, err = url.Parse(referer); err != nil {
			return &url.URL{}, err
		}
	}
	var tmp = raw
	// b) no scheme, no host, relative path (causes empty scheme/host result)
	//    (are dot paths allowed in links?)
	if strings.HasPrefix(raw, "/") {
		tmp = fmt.Sprintf("gemini://%s%s", rfr.Host, raw)
	} else if foundAt := strings.Index(raw, ":/"); foundAt == -1 {
		// a) no scheme, host without port (causes empty Parse result)
		tmp = "gemini://" + raw
	}

	u, err := url.Parse(tmp)
	if err != nil {
		return &url.URL{}, fmt.Errorf("Error parsing URL! %v", err)
	}
	if !u.IsAbs() {
		// relative?
		u.Scheme = rfr.Scheme
		if u.Hostname() == "" {
			//is-relative
			u.Host = rfr.Host
		}
	}
	if u.Port() == "" && u.Scheme == "gemini" {
		u.Host += ":1965"
	}

	return u, nil
}

func NewControl(ctx context.Context) *control {
	var ctrl = &control{
		rules: safemap{m: make(map[string]*rewriter)},
		ctx:   ctx,
	}

	ctrl.Attach(GmPlain, vanilla)
	ctrl.Attach(GmLink, rewriteLink)
	return ctrl
}

func (c *control) Dial(u *url.URL) (io.Reader, error) {
	var err error
	c.conn, err = tls.Dial("tcp", u.Host, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %v", err)
	}
	c.state = CONNECTION_OPEN

	// Send request (CR LF terminated)
	c.conn.Write([]byte(u.String() + "\r\n"))

	// Receive and parse response header
	var reader = bufio.NewReader(c.conn)
	responseHeader, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Failed to read response %v", err)
	}
	var parts = strings.Fields(responseHeader)
	status, err := strconv.Atoi(parts[0][0:1])
	if err != nil {
		return nil, fmt.Errorf("Failed to extract status %v", err)
	}
	var meta = parts[1]

	switch status {
	case 1, 6:
		// No input, or client certs
		return nil, fmt.Errorf("Unsupported feature! - %v", status)

	case 2:
		// Successful transaction
		// text/* content only
		if !strings.HasPrefix(meta, "text/") {
			return nil, fmt.Errorf("Unsupported type %s", meta)
		}
		return reader, nil

	case 3:
		// TODO use config setting to follow redirects (enable-follow as default)
		return nil, fmt.Errorf("Not-implemented: REDIR %v", meta)
	case 4, 5:
		return nil, fmt.Errorf("ERROR: %v", meta)
	}

	return nil, fmt.Errorf("Exceptional status code did not match known values.")
}

// Disconnect and close gr channels
func (c *control) Close() {
	c.rules.Lock()
	defer c.rules.Unlock()

	if c.state != CONNECTION_CLOSE {
		//todo atomic set
		c.state = CONNECTION_CLOSE
		c.conn.Close()
	}
	for _, run := range c.rules.m {
		close(run.ch)
	}
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

func (c *control) Retrieve(r io.Reader) (string, error) {
	c.rules.Lock()
	defer c.rules.Unlock()

	var (
		buffer = make(chan string)
		bld    strings.Builder
	)
	// grab the entire gemini body since Parse() accepts the body as string
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	tree, err := Parse(string(b))
	if err != nil {
		return "", err
	}

	//TODO support cancellable
	grp, _ := errgroup.WithContext(c.ctx)
	go func() {
		// accumulate results
		for row := range buffer {
			bld.WriteString(row)
		}
	}()
	// tree walk
	for _, n := range tree.Root.Nodes {
		switch n.Type() {
		case NodeLink:
			if run, ok := c.rules.m[GmLink]; ok {
				spawn(run.ch, run.fn, buffer, grp)
				run.ch <- n
			}
		default:
			if run, ok := c.rules.m[GmPlain]; ok {
				spawn(run.ch, run.fn, buffer, grp)
				run.ch <- n
			}
		}
	}

	// wait for grs to complete
	grp.Wait()
	// signal the for/range to end
	close(buffer)

	return bld.String(), nil
}
