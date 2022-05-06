// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"image"
	"image/color"
)
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

const VScrollBarWidth = 1

type Scroll struct {
	X      int
	Y      int
	Height int
	Rect   image.Rectangle
	fg     color.RGBA

	thumbRate           float64
	thumbOffset         int
	dragging            bool
	draggingStartOffset int
	draggingStartY      int
	contentOffset       int
}

func newScroll(wd, ht int) *Scroll {
	var x = wd - VScrollBarWidth
	var y = pxht * 2
	var vht = ht - y - 20
	var bgxy = image.Rect(x, y, wd, ht-20)

	return &Scroll{
		Rect:   bgxy,
		fg:     color.RGBA{0x66, 0x33, 0x99, 0xff},
		Height: vht,
		X:      x,
		Y:      y,
	}
}

func (v *Scroll) thumbSize() int {
	const minThumbSize = VScrollBarWidth

	r := v.thumbRate
	if r > 1 {
		r = 1
	}
	s := int(float64(v.Height) * r)
	if s < minThumbSize {
		return minThumbSize
	}
	return s
}

func (v *Scroll) thumbRect() image.Rectangle {
	if v.thumbRate >= 1 {
		return image.Rectangle{}
	}
	//todo wishlist is gradient
	var thick = 4 * VScrollBarWidth
	var s = v.thumbSize()
	return image.Rect(v.X-thick, v.Y+v.thumbOffset, v.X, v.Y+v.thumbOffset+s)
}

func (v *Scroll) maxThumbOffset() int {
	return v.Height - v.thumbSize()
}

func (v *Scroll) ContentOffset() int {
	return v.contentOffset
}

func (v *Scroll) Update(contentHeight int) {
	v.thumbRate = float64(v.Height) / float64(contentHeight)

	if !v.dragging && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		var x, y = ebiten.CursorPosition()
		var tr = v.thumbRect()
		if tr.Min.X <= x && x < tr.Max.X && tr.Min.Y <= y && y < tr.Max.Y {
			v.dragging = true
			v.draggingStartOffset = v.thumbOffset
			v.draggingStartY = y
		}
	}
	if v.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			v.thumbOffset = v.draggingStartOffset + (y - v.draggingStartY)
			if v.thumbOffset < 0 {
				v.thumbOffset = 0
			}
			if v.thumbOffset > v.maxThumbOffset() {
				v.thumbOffset = v.maxThumbOffset()
			}
		} else {
			v.dragging = false
		}
	}

	v.contentOffset = 0
	if v.thumbRate < 1 {
		v.contentOffset = int(float64(contentHeight) * float64(v.thumbOffset) / float64(v.Height))
	}
}

func (v *Scroll) Draw(screen *ebiten.Image) {
	var bg = screen.SubImage(v.Rect).(*ebiten.Image)
	bg.Fill(color.RGBA{0x66, 0x33, 0x99, 0x80}) // 0x80 to try 50% alpha
	if v.thumbRate < 1 {
		var mark = screen.SubImage(v.thumbRect()).(*ebiten.Image)
		mark.Fill(v.fg)
	}
}
