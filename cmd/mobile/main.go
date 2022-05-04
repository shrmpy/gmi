package main

import (
	"errors"
	"image/color"
	"log"
)

import "github.com/hajimehoshi/ebiten/v2"

type Game struct {
	panel *Panel
	bus   chan signal
}

func (g *Game) Layout(w int, h int) (int, int) { return w, h }
func (g *Game) Update() error {
	select {
	case req := <-g.bus:
		if req.op == 8888 {
			// the quit signal
			return errors.New("teardown")
		}
		if req.op == 1965 {
			g.panel.Reset()
			g.geminiPod(req.data)
		}
	default:
		g.panel.Update()
	}

	return nil
}
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x00, 0x00, 0x00, 0xff})
	g.panel.Draw(screen)
}

func main() {
	var (
		ht = 640
		wd = 360
		pn = newPanel(wd, ht)
		ch = make(chan signal, 100)
	)
	defer close(ch)
	pn.QuitFunc(func(el Element) {
		ch <- signal{op: 8888}
	})
	var gm = &Game{panel: pn, bus: ch}
	pn.GeminiFunc(gm.geminiPod)

	ebiten.SetWindowTitle("gmimo")
	ebiten.SetWindowSize(wd, ht)
	var err = ebiten.RunGame(gm)
	if err != nil {
		log.Fatal(err)
	}
}

type signal struct {
	op   int
	data string
}
