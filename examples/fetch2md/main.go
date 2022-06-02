package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
)
import "github.com/shrmpy/gmi"

func main() {
	var cp = flag.String("cap", "gemini://gemini.circumlunar.space", "Capsule address")
	flag.Parse()

	var cfg = &config{}
	var md = transform(*cp, cfg)
	fmt.Println(md)
}
func transform(capsule string, cfg *config) string {
	var (
		err error
		req *url.URL
		rdr *bufio.Reader
		md  string
	)
	var ctrl = gmi.NewControl(context.Background())
	ctrl.Attach(gmi.GmLink, rewriteLink)
	ctrl.Attach(gmi.GmPlain, rewritePlain)
	if req, err = gmi.Format(capsule, ""); err != nil {
		log.Fatalf("DEBUG Capsule URL, %v", err)
	}
	if rdr, err = ctrl.Dial(req, cfg); err != nil {
		log.Fatalf("DEBUG Dial, %v", err)
	}
	defer ctrl.Close()
	if md, err = ctrl.Retrieve(rdr); err != nil {
		log.Fatalf("DEBUG Retrieve, %v", err)
	}
	return md
}
func rewriteLink(n gmi.Node) string {
	var lnk = n.(*gmi.LinkNode)
	var name = lnk.Friendly
	var lu = lnk.URL
	if name == "" {
		name = lu.String()
	}
	// markdown of hyperlink
	return fmt.Sprintf("[=> %s](%s)\n", name, lu)
}
func rewritePlain(n gmi.Node) string {
	return fmt.Sprintf("%s\n", n)
}

type config struct{}

func (c *config) ISV() gmi.Mask {
	return gmi.AcceptUAE | gmi.AcceptLCN
}
func (c *config) KnownHosts() string {
	return "known_capsules"
}
