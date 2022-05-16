package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"sort"
	"strings"
)

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"
import "github.com/tinne26/etxt"

//go:embed NotoSansMono-Regular.ttf
var notoSansMonoTTF []byte

//go:embed DejaVuSansMono.ttf
var dejavuSansMonoTTF []byte

////const monospaceFont = "Noto Sans Mono Regular"
const monospaceFont = "DejaVu Sans Mono"
const pxht = 18

// text page widget
type Panel struct {
	txtRenderer         *etxt.Renderer
	sinceLastSpecialKey int
	bar                 *Bar
	scroll              *Scroll
	burger              *Icon
	fg                  color.RGBA
	lines               []*GemLine
	gemini              func(string)
	sorted              bool
	offsetY             int
	wd                  int
	ht                  int
	contentBuf          *ebiten.Image
	fonts               *etxt.FontLibrary
}

func (p *Panel) Update() error {
	p.burger.Update()
	p.scroll.Update(p.contentSize())
	p.offsetY = p.scroll.ContentOffset()
	p.gemActions()
	p.bar.Update()
	return nil
}
func (p *Panel) Draw(screen *ebiten.Image) {
	p.txtRenderer.SetTarget(screen)
	p.txtRenderer.SetColor(p.fg)

	p.drawLines(screen)

	p.bar.Draw(screen, p.txtRenderer)
	p.scroll.Draw(screen)
	p.burger.Draw(p.txtRenderer)
}
func (p *Panel) drawLines(screen *ebiten.Image) {
	if !p.sorted {
		return
	}
	// is the drawing buffer for performance?
	var buf = p.buffer()
	p.txtRenderer.SetTarget(buf)
	defer p.txtRenderer.SetTarget(screen)

	var lineHt = pxht
	var x = 0
	var y = -p.offsetY
	_, vht := p.viewSize()

	for i, line := range p.lines {
		if i != 0 {
			//TODO wrapping adjustment
			y += lineHt
		}
		if y < -lineHt {
			continue
		} // above viewport
		if y >= vht+lineHt {
			continue
		} // below viewport
		line.Draw(p.txtRenderer, x, y)
	}

	var op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(lineHt))
	screen.DrawImage(buf, op)
}
func (p *Panel) gemActions() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	var x, y = ebiten.CursorPosition()
	if 1 > x || x >= p.scroll.X ||
		pxht > y || y >= p.ht-20 {
		return
	}
	// click is inside page bounds
	var (
		acc = pxht
		// y position that is inside viewport
		vy = y
	)
	for i, line := range p.lines {
		if acc < p.offsetY {
			// still off-screen
			acc += line.Height()
			continue
		}
		// now lines are in viewport
		var prev = acc - p.offsetY
		var next = acc + line.Height() - p.offsetY
		if vy >= prev && vy < next {
			p.lines[i].Action()
			break
		}
		acc += line.Height()
	}
}
func (p *Panel) AppendLink(sequence int, name string, lu string, f func(addr string)) {
	var r = &GemLine{
		Sequence: sequence,
		LinkURL:  lu,
	}
	r.Icon.fg = color.RGBA{0xad, 0xff, 0x2f, 0xff}
	r.Icon.Text = p.tofu(name)
	r.Icon.HandleFunc(func(el Element) {
		f(r.LinkURL)
	})
	p.lines = append(p.lines, r)
}
func (p *Panel) AppendParagraph(sequence int, text string) {
	var r = &GemLine{Sequence: sequence}
	r.Icon.fg = color.RGBA{0xff, 0xff, 0xff, 0xff}
	r.Icon.Text = p.tofu(text)
	p.lines = append(p.lines, r)
}
func (p *Panel) QuitFunc(f func(el Element)) {
	// accept callback function to attach to burger icon
	// (which quits program)
	p.burger.HandleFunc(f)
}
func (p *Panel) GeminiFunc(f func(addr string)) {
	// avoid making bus into global scope
	// container has bus channel so its function
	// is attached for signaling when Gem links
	// are activated
	p.gemini = f
	p.bar.HandleFunc(func(el Element) {
		var lu = el.(*Icon).Text
		if foundAt := strings.Index(lu, ":/"); foundAt == -1 {
			// less ambiguous for URL formatter
			lu = fmt.Sprintf("gemini://%s", lu)
		}
		p.Reset()
		p.gemini(lu)
	})
}
func (p *Panel) Skip() { p.sorted = false }
func (p *Panel) Resume() {
	// perform sort as goroutines can append lines out of sequence
	sort.Slice(p.lines, func(i, j int) bool {
		return p.lines[i].Sequence < p.lines[j].Sequence
	})
	p.sorted = true
}
func (p *Panel) Reset() {
	// pre-process to launching link
	// wipe the page lines
	p.Skip()
	p.lines = nil
	p.Resume()
}
func (p *Panel) contentSize() int {
	//TODO wrapping lines
	var sz = len(p.lines) * pxht
	return sz
}
func (p *Panel) viewSize() (int, int) {
	// static viewport for now
	// (leave margins for 1 line at the top
	// and bar at the bottom)
	var vert = p.ht - pxht - 20
	return p.wd, vert
}
func (p *Panel) buffer() *ebiten.Image {
	// we assume the static viewport
	// to simplify the drawing buffer
	if p.contentBuf == nil {
		var w, h = p.viewSize()
		p.contentBuf = ebiten.NewImage(w, h)
	}
	p.contentBuf.Clear()
	return p.contentBuf
}
func (p *Panel) tofu(text string) string {
	//TODO maybe we need to extend the renderer
	//     to deal with (more intl) unicode;
	//     naive idea, switch fonts (worst case is rune by rune)
	var (
		miss []rune
		err  error
		tmp  = text
		font = p.fonts.GetFont(monospaceFont)
	)
	miss, err = etxt.GetMissingRunes(font, text)
	if err == nil {
		// substitute missing runes with tofu (◻)
		for _, char := range miss {
			tmp = strings.ReplaceAll(tmp, string(char), "◻")
		}
	}
	return tmp
}

func newPanel(wd, ht int) *Panel {
	var (
		err   error
		name  string
		fonts = etxt.NewFontLibrary()
		fg    = color.RGBA{0x66, 0x33, 0x99, 0xff} // pnl foreground text
	)
	name, err = fonts.ParseFontBytes(notoSansMonoTTF)
	if err != nil {
		log.Fatalf("INFO Parse error Noto Sans Mono, %v", err.Error())
	}
	log.Printf("+ font %s", name)
	name, err = fonts.ParseFontBytes(dejavuSansMonoTTF)
	if err != nil {
		log.Fatalf("INFO Parse error DejaVu Sans Mono, %v", err.Error())
	}
	log.Printf("+ font %s", name)
	var font = fonts.GetFont(monospaceFont)
	var cache = etxt.NewDefaultCache(4 * 1024 * 1024) // 4MB cache

	// create and configure renderer
	var renderer = etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(pxht)
	renderer.SetFont(font)

	var burg = newBurger(wd, ht, renderer, fg)
	var bar = newBar(wd, ht)
	var scroll = newScroll(wd, ht)

	return &Panel{
		txtRenderer: renderer,
		bar:         bar,
		burger:      burg,
		scroll:      scroll,
		fg:          fg,
		fonts:       fonts,
		wd:          wd,
		ht:          ht,
	}
}

type GemLine struct {
	Icon     // composition of basic element
	Sequence int
	LinkURL  string
}

func (r *GemLine) Draw(renderer *etxt.Renderer, x int, y int) {
	r.Icon.x = x
	r.Icon.y = y
	r.Icon.Draw(renderer)
}
func (r *GemLine) Action() error {
	return r.Icon.Action()
}
func (r *GemLine) HandleFunc(f func(el Element)) {
	r.Icon.HandleFunc(f)
}
func (r *GemLine) Height() int {
	//TODO wrapping adjustment

	return pxht
}

type Bar struct {
	Icon                // nothing shared?
	Rect                image.Rectangle
	fg                  color.RGBA
	sinceLastSpecialKey int
	text                []rune
}

func (b *Bar) Update() {
	backspacePressed := ebiten.IsKeyPressed(ebiten.KeyBackspace)
	enterPressed := ebiten.IsKeyPressed(ebiten.KeyEnter)

	//TODO with the cmd bar shift/pan the text left and rt
	if backspacePressed && b.sinceLastSpecialKey >= 7 && len(b.text) >= 1 {
		b.sinceLastSpecialKey = 0
		b.text = b.text[0 : len(b.text)-1]
	} else if enterPressed && b.sinceLastSpecialKey >= 20 {
		b.sinceLastSpecialKey = 0
		b.Action()
	} else {
		b.sinceLastSpecialKey += 1
		b.text = ebiten.AppendInputChars(b.text)
	}
}

func (b *Bar) Draw(screen *ebiten.Image, renderer *etxt.Renderer) {
	renderer.SetAlign(etxt.Bottom, etxt.Left)
	var bg = screen.SubImage(b.Rect).(*ebiten.Image)
	bg.Fill(color.RGBA{0x66, 0x33, 0x99, 0x80}) // 0x80 to try 50% alpha
	renderer.SetColor(b.fg)
	renderer.Draw(string(b.text), 0, b.Rect.Max.Y)
}
func (b *Bar) Action() error {
	b.Icon.Text = string(b.text)
	return b.Icon.Action()
}
func (b *Bar) HandleFunc(f func(el Element)) {
	b.Icon.HandleFunc(f)
}
func newBar(wd, ht int) *Bar {
	var bgxy = image.Rect(1, ht-20, wd-1, ht-1)
	return &Bar{
		Rect: bgxy,
		text: []rune("Type URL"),
		fg:   color.RGBA{0xff, 0xff, 0xff, 0xff},
	}
}
func newBurger(wd, ht int, renderer *etxt.Renderer, fg color.RGBA) *Icon {
	var label = "≡"
	var sz = renderer.SelectionRect(label)
	return newIcon(wd-pxht, 0, etxt.Top, etxt.Left, label, sz, fg)
}

// icon is the burger so far (skeleton element)
type Icon struct {
	x, y      int
	va        etxt.VertAlign
	ha        etxt.HorzAlign
	Text      string
	mouseDown bool
	rectSize  etxt.RectSize
	onPressed func(el Element)
	fg        color.RGBA
}

func newIcon(x, y int, va etxt.VertAlign, ha etxt.HorzAlign,
	label string, sz etxt.RectSize, fg color.RGBA) *Icon {
	return &Icon{
		x:        x,
		y:        y,
		va:       va,
		ha:       ha,
		Text:     label,
		rectSize: sz,
		fg:       fg,
	}
}
func (i *Icon) Update() {
	var click = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if !click {
		if i.mouseDown {
			i.Action()
		}
		i.mouseDown = false
		return
	}
	//todo align v/h affects the rect size orientation
	var (
		minx = i.x
		miny = i.y
		maxx = i.x + i.rectSize.WidthCeil()
		maxy = i.y + i.rectSize.HeightCeil()
	)
	var x, y = ebiten.CursorPosition()
	if minx <= x && x < maxx && miny <= y && y < maxy {
		// calc cursor lands in box
		i.mouseDown = true
	} else {
		i.mouseDown = false
	}
}
func (i *Icon) Draw(renderer *etxt.Renderer) {
	renderer.SetAlign(i.va, i.ha)
	renderer.SetColor(i.fg)
	renderer.Draw(i.Text, i.x, i.y)
}
func (i *Icon) Action() error {
	if i.onPressed != nil {
		i.onPressed(i)
	}
	return nil
}
func (i *Icon) HandleFunc(f func(el Element)) {
	i.onPressed = f
}

// the minimum UI element is text that responds to events
type Element interface {
	Action() error
	HandleFunc(f func(el Element))
}
