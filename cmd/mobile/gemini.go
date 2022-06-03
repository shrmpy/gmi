package main

import (
	"bufio"
	"context"
	"log"
	"net/url"

	"github.com/shrmpy/gmi"
)

func (g *Game) capsule(addr string) {
	var (
		req *url.URL
		rdr *bufio.Reader
		err error
		ctx = context.Background()
	)
	// avoid coupling gmi pkg to cfg struct
	var ctrl = gmi.NewControl(ctx)
	// substitute our customized rules
	ctrl.Attach(gmi.LinkLine, g.rewriteLink)
	ctrl.Attach(gmi.PlainLine, g.rewritePlain)
	log.Printf("INFO Format URL, %s", addr)
	if req, err = gmi.Format(addr, string(g.panel.bar.text)); err != nil {
		log.Printf("INFO URL format error, %v", err.Error())
		return
	}
	var params = &geminiParams{args: g.cfg}
	log.Printf("INFO Dial Gemini pod, %s", req.String())
	if rdr, err = ctrl.Dial(req, params); err != nil {
		log.Printf("INFO Dial error, %v", err.Error())
		return
	}
	defer ctrl.Close()
	log.Printf("INFO Draw paused")
	g.panel.Skip()
	defer g.panel.Resume()
	log.Printf("INFO Gemini content, %d", rdr.Buffered())
	if _, err = ctrl.Retrieve(rdr); err != nil {
		log.Printf("INFO Retrieve error, %v", err.Error())
		return
	}
	//todo setter
	g.panel.bar.text = []rune(req.String())
	log.Printf("INFO Draw resumed")
}

// define how to treat Gem links
func (g *Game) rewriteLink(no gmi.Node) string {
	var (
		lnk  = no.(*gmi.LinkNode)
		seq  = lnk.Position()
		lu   = lnk.URL.String()
		name = lnk.Friendly
	)
	if lnk.Friendly == "" {
		name = lu
	}
	log.Printf("INFO Gem link pos %d, %s", seq, lu)
	g.panel.AppendLink(int(seq), name, lu, func(addr string) {
		// send signal to the dispatcher
		// it contains the link URL as the data field
		g.bus <- signal{op: 1965, data: addr}
	})
	return ""
}

// define how to treat Gem plain text
func (g *Game) rewritePlain(no gmi.Node) string {
	var (
		tn  = no.(*gmi.TextNode)
		seq = tn.Position()
	)
	log.Printf("INFO Gem plain pos %d", seq)
	g.panel.AppendParagraph(int(seq), no.String())
	return ""
}
