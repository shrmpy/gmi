package main

import (
	"fmt"
	"image/color"
	"log"
)

import "github.com/hajimehoshi/ebiten/v2"

type Game struct {
	panel *Panel
	bus   chan signal
	cfg   *argsCfg
}

func (g *Game) Layout(w int, h int) (int, int) { return w, h }
func (g *Game) Update() error {
	select {
	case req := <-g.bus:
		if req.op == 8888 {
			// the quit signal
			var er = fmt.Errorf("teardown")
			log.Printf("INFO Program exiting by burger icon, %v ", er.Error())
			return er
		}
		if req.op == 1965 {
			g.panel.Reset()
			g.capsule(req.data)
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
	var cfg, err = readArgs()
	if err != nil {
		log.Fatalf("ERROR Config arguments, %v", err.Error())
	}
	pn.QuitFunc(func(el Element) {
		ch <- signal{op: 8888}
	})
	var gm = &Game{panel: pn, bus: ch, cfg: cfg}
	pn.GeminiFunc(gm.capsule)

	ebiten.SetWindowTitle("gmimo")
	ebiten.SetWindowSize(wd, ht)
	if err = ebiten.RunGame(gm); err != nil {
		//todo custom error type
		log.Fatalf("ERROR Quit, %v ", err)
	}
}

type signal struct {
	op   int
	data string
}
