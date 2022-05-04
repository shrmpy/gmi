package main

import (
	"context"

	"github.com/shrmpy/gmi"
)

func (a *container) geminiPod(url string, referer string) {
	var ctrl = gmi.NewControl(context.Background())
	// substitute our custom rules
	ctrl.Attach(gmi.GmLink, a.rewriteLink)
	ctrl.Attach(gmi.GmPlain, a.rewritePlain)

	req, err := gmi.Format(url, referer)
	if err != nil {
		a.status.SetRight(err.Error())
		return
	}
	rdr, err := ctrl.Dial(req)
	if err != nil {
		a.status.SetRight(err.Error())
		return
	}
	defer ctrl.Close()
	// beginning page change, temporarily postpone its drawing
	a.gvw.Skip()
	// fetch gemini content (and trigger rules)
	_, err = ctrl.Retrieve(rdr)
	if err != nil {
		a.status.SetRight(err.Error())
	}
	a.bag.url = req.String()
	a.gvw.Resume()
}

// define how to treat Gem links
func (a *container) rewriteLink(n gmi.Node) string {
	var lnk *gmi.LinkNode
	lnk = n.(*gmi.LinkNode)
	var (
		seq  = lnk.Position()
		ur   = lnk.URL.String()
		name = lnk.Friendly
	)
	if name == "" {
		name = ur
	}
	a.gvw.AppendLink(int(seq), name, ur, func(u string) {
		// send signal to the dispatcher
		// it contains the link URL as the data field
		a.bus <- signal{op: 1965, data: u}
	})
	return ""
}

// define how to treat Gem plain text
func (a *container) rewritePlain(n gmi.Node) string {
	var (
		tn  = n.(*gmi.TextNode)
		seq = tn.Position()
	)
	a.gvw.AppendParagraph(int(seq), n.String())
	return ""
}
