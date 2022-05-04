package main

import (
	"context"
	"net/url"

	"github.com/shrmpy/gmi"
)

func (g *Game) geminiPod(addr string) {
	var (
		req  *url.URL
		err  error
		ctrl = gmi.NewControl(context.Background())
	)
	// substitute our customized rules
	ctrl.Attach(gmi.GmLink, g.rewriteLink)
	ctrl.Attach(gmi.GmPlain, g.rewritePlain)

	if req, err = gmi.Format(addr, string(g.panel.bar.text)); err != nil {
		g.panel.AppendParagraph(1,
			"DEBUG URL format error "+err.Error())
		return
	}
	rdr, err := ctrl.Dial(req)
	if err != nil {
		g.panel.AppendParagraph(1,
			"DEBUG dial error "+err.Error())
		return
	}
	defer ctrl.Close()
	// beginning page change, temporarily postpone its drawing
	g.panel.Skip()
	// fetch gemini content (and trigger rules)
	_, err = ctrl.Retrieve(rdr)
	if err != nil {
		g.panel.AppendParagraph(1,
			"DEBUG retrv error "+err.Error())
	}
	g.panel.bar.text = []rune(req.String())
	g.panel.Resume()
}

// define how to treat Gem links
func (g *Game) rewriteLink(n gmi.Node) string {
	var (
		lnk  = n.(*gmi.LinkNode)
		seq  = lnk.Position()
		lu   = lnk.URL.String()
		name = lnk.Friendly
	)
	if lnk.Friendly == "" {
		name = lu
	}

	g.panel.AppendLink(int(seq), name, lu, func(addr string) {
		// send signal to the dispatcher
		// it contains the link URL as the data field
		g.bus <- signal{op: 1965, data: addr}
	})
	return ""
}

// define how to treat Gem plain text
func (g *Game) rewritePlain(n gmi.Node) string {
	var (
		tn  = n.(*gmi.TextNode)
		seq = tn.Position()
	)

	g.panel.AppendParagraph(int(seq), n.String())
	return ""
}
