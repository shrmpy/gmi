// Copyright 2016 The Tcell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

var app *views.Application

type container struct {
	gvw    *GemView
	keybar *views.SimpleStyledText
	status *views.SimpleStyledTextBar
	bag    *gembag
	bus    chan signal

	views.Panel
}

func (a *container) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlL:
			app.Refresh()
			return true
		case tcell.KeyEnter:
			if a.bag.gemini {
				a.bag.gemini = false
				a.bus <- signal{op: 1965, data: a.bag.url}
			} else {
				// certain lines can be clicked
				a.gvw.Actions()
			}
			// call redraw step
			a.updateKeys()
			return true
		case tcell.KeyBackspace, tcell.KeyDelete, tcell.KeyBackspace2:
			var size = len(a.bag.url)
			if a.bag.gemini && size > 0 {
				a.bag.url = a.bag.url[:size-1]
				return true
			}

		case tcell.KeyRune:
			if a.bag.gemini {
				a.bag.url += string(ev.Rune())
				return true
			}
			switch ev.Rune() {
			case 'Q', 'q':
				a.bus <- signal{op: 8888}
				return true
			case 'S', 's':
				a.gvw.HideCursor(false)
				a.updateKeys()
				return true
			case 'H', 'h':
				a.gvw.HideCursor(true)
				a.updateKeys()
				return true
			case 'E', 'e':
				a.gvw.EnableCursor(true)
				a.updateKeys()
				return true
			case 'D', 'd':
				a.gvw.EnableCursor(false)
				a.updateKeys()
				return true
			case ':':
				a.bag.gemini = true
				a.updateKeys()
				return true
			}
		}
	}
	return a.Panel.HandleEvent(ev)
}
func (a *container) Draw() {
	select {
	case req := <-a.bus:
		if req.op == 1965 {
			// launch link URL signal
			a.geminiPod(req.data, a.bag.url)
			a.status.SetCenter(a.bag.url)

		} else if req.op == 8888 {
			// the shutdown signal
			app.Quit()
		}
	default:
		if a.bag.gemini {
			a.status.SetLeft("gemini://")
			a.status.SetCenter(a.bag.url)
		} else {
			a.status.SetLeft(a.bag.loc)
		}
		a.Panel.Draw()
	}
}

// "rebind" menu-bar
func (a *container) updateKeys() {
	var (
		mo = a.gvw.GetModel()
		mb = "[%AQ%N] Quit"
	)
	_, _, enab, shown := mo.GetCursor()
	if !enab {
		mb += "  [%AE%N] Enable cursor"
	} else {
		mb += "  [%AD%N] Disable cursor"
		if shown {
			mb += "  [%AH%N] Hide cursor"
		} else {
			mb += "  [%AS%N] Show cursor"
		}
	}
	a.keybar.SetMarkup(mb)
	app.Update()
}

func main() {
	var ch = make(chan signal, 100)
	defer close(ch)
	app = &views.Application{}

	var parent = &container{
		bus: ch,
		bag: &gembag{endx: 60, endy: 15},
	}

	parent.keybar = views.NewSimpleStyledText()
	parent.keybar.RegisterStyle('N', tcell.StyleDefault.
		Background(tcell.ColorRebeccaPurple).
		Foreground(tcell.ColorBlack))
	parent.keybar.RegisterStyle('A', tcell.StyleDefault.
		Background(tcell.ColorRebeccaPurple).
		Foreground(tcell.ColorLime))

	parent.status = views.NewSimpleStyledTextBar()
	parent.status.SetStyle(tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorSilver))
	parent.status.RegisterLeftStyle('N', tcell.StyleDefault.
		Background(tcell.ColorDarkViolet).
		Foreground(tcell.ColorHoneydew))

	parent.status.SetLeft("My status is here.")
	parent.status.SetRight("demo!")
	parent.status.SetCenter("Type colon to input URL")
	var title = views.NewTextBar()
	title.SetStyle(tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite))
	title.SetCenter("reader", tcell.StyleDefault)
	title.SetRight("gmit v0.0.9", tcell.StyleDefault)

	parent.gvw = NewGemView()

	parent.SetMenu(parent.keybar)
	parent.SetTitle(title)
	parent.SetContent(parent.gvw)
	parent.SetStatus(parent.status)

	parent.updateKeys()

	app.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack))
	app.SetRootWidget(parent)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

type gembag struct {
	x    int
	y    int
	endx int
	endy int

	loc    string
	gemini bool
	url    string
}
type signal struct {
	op   int
	data string
}
