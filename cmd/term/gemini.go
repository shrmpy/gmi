package main

import (
	"context"
	//"fmt"

	"github.com/shrmpy/gmi"
)

func (a *container) capsule(url string, referer string) {
	var ctrl = gmi.NewControl(context.Background())
	// substitute our custom rules
	ctrl.Attach(gmi.GmLink, a.rewriteLink)
	ctrl.Attach(gmi.GmPlain, a.rewritePlain)

	req, err := gmi.Format(url, referer)
	if err != nil {
		a.status.SetRight(err.Error())
		return
	}
	var params = &geminiParams{args: a.cfg}
	rdr, err := ctrl.Dial(req, params)
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
	var (
		lnk  = n.(*gmi.LinkNode)
		seq  = lnk.Position()
		lu   = lnk.URL.String()
		name = lnk.Friendly
	)
	if name == "" {
		name = lu
	}
	//DEBUG
	////name = fmt.Sprintf("%s %s %s", lnk.URL.Scheme, lnk.URL.Host, lnk.URL.EscapedPath())
	//DEBUG

	a.gvw.AppendLink(int(seq), name, lu, func(u string) {
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
