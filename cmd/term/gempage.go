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
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// GemView is the cell-view panel that the main window will draw
// (encapsulate the cell-model from render concerns)
// - sorted flag, true indicates page lines finished update
// - scratch is a throw-away temp space for page lines update
type GemView struct {
	views.CellView
	sorted  bool
	scratch []*GemLine
}

func (p *GemView) AppendLink(sequence int, name string, lu string, f func(u string)) {
	if p.sorted {
		// sanity check (enforce Skip() is called first)
		return
	}
	var li = &GemLine{
		Sequence: sequence,
		Text:     name,
		LinkURL:  lu,
	}
	li.style = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorGreen).
		Underline(true)
	// attach callback/fn for line-click event
	li.SetOnPressed(func(th *GemLine) {
		f(th.LinkURL)
	})
	p.scratch = append(p.scratch, li)
}
func (p *GemView) AppendParagraph(sequence int, text string) {
	if p.sorted {
		// sanity check (enforce Skip() is called first)
		return
	}
	var li = &GemLine{
		Sequence: sequence,
		Text:     text,
	}
	p.scratch = append(p.scratch, li)
}

// skip render step for page lines
func (p *GemView) Skip() {
	p.sorted = false
	p.scratch = []*GemLine{}
}

// resume render step for page lines
func (p *GemView) Resume() {
	// perform sort as goroutines can append lines out of sequence
	sort.Slice(p.scratch, func(i, j int) bool {
		return p.scratch[i].Sequence < p.scratch[j].Sequence
	})
	// calc width of cell-model
	var w int
	for _, line := range p.scratch {
		//todo a rune is not always byte size
		var count = len(line.Text)
		if w < count {
			w = count
		}
	}

	var m = p.CellView.GetModel().(*GemPage)
	m.width = w
	m.height = len(p.scratch)
	m.lines = make([]*GemLine, m.height)
	copy(m.lines, p.scratch)

	// now it's safe to modify the model (with SetLines())
	p.CellView.SetModel(m)
	p.sorted = true
}
func (p *GemView) Actions() {
	//TODO determine cursor x,y and whether it falls on link line
	var m = p.CellView.GetModel().(*GemPage)
	_, cy, en, sh := m.GetCursor()
	if !en || !sh {
		// cursor is not enabled
		return
	}
	//todo panning offset calc
	var l = m.lines[cy]
	l.Action()
}

// EnableCursor enables a soft cursor in the TextArea.
func (p *GemView) EnableCursor(on bool) {
	var m = p.GetModel().(*GemPage)
	m.cursor = on
	p.SetModel(m)
}

// HideCursor hides or shows the cursor in the TextArea.
// If on is true, the cursor is hidden.  Note that a cursor is only
// shown if it is enabled.
func (p *GemView) HideCursor(on bool) {
	var m = p.GetModel().(*GemPage)
	m.hide = on
	p.SetModel(m)
}

// NewGemView creates a blank TextArea.
func NewGemView() *GemView {
	var p = &GemView{}
	var m = &GemPage{width: 0}

	p.CellView.Init()
	p.CellView.SetModel(m)
	return p
}

// provides a CellModel interface
type GemPage struct {
	lines  []*GemLine
	width  int
	height int
	x      int
	y      int
	hide   bool
	cursor bool
}

func (m *GemPage) GetCell(x, y int) (rune, tcell.Style, []rune, int) {
	if x < 0 || y < 0 || y >= m.height || x >= len([]rune(m.lines[y].Text)) {
		return 0, tcell.StyleDefault, nil, 1
	}
	//DEBUG
	var line = m.lines[y]
	var runes = []rune(line.Text)
	return runes[x], line.style, nil, 1
}
func (m *GemPage) GetBounds() (int, int) {
	return m.width, m.height
}
func (m *GemPage) limitCursor() {
	if m.x > m.width-1 {
		m.x = m.width - 1
	}
	if m.y > m.height-1 {
		m.y = m.height - 1
	}
	if m.x < 0 {
		m.x = 0
	}
	if m.y < 0 {
		m.y = 0
	}
}
func (m *GemPage) SetCursor(x, y int) {
	m.x = x
	m.y = y
	m.limitCursor()
}
func (m *GemPage) MoveCursor(x, y int) {
	m.x += x
	m.y += y
	m.limitCursor()
}
func (m *GemPage) GetCursor() (int, int, bool, bool) {
	return m.x, m.y, m.cursor, !m.hide
}

type GemLine struct {
	Sequence  int
	Text      string
	LinkURL   string
	onPressed func(l *GemLine)
	style     tcell.Style
}

// assign callback/fn, during page's node tree init
func (l *GemLine) SetOnPressed(f func(l *GemLine)) {
	l.onPressed = f
}

// used by page to trigger line's action
func (l *GemLine) Action() {
	if l.onPressed == nil {
		return
	}
	// signal page to load URL
	// (page container would have attached fn earlier)
	l.onPressed(l)
}
