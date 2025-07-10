package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	cityWindowWidth  = 400
	cityWindowHeight = 600
)

func (g *Game) initCityWindow() {
	g.cityWindow = &CityWindow{
		game: g,
		open: false,
	}
}

func (w *CityWindow) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		w.open = !w.open
	}

	if !w.open {
		return
	}

	_, dy := ebiten.Wheel()
	w.scrollY += int(dy * 20)
	if w.scrollY < 0 {
		w.scrollY = 0
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= 0 && mx <= cityWindowWidth && my >= 0 && my <= cityWindowHeight {
			index := (my + w.scrollY - 50) / 40
			if index >= 0 && index < len(w.game.cityList) {
				w.selected = w.game.cityList[index]
			}
		}
	}

	mx, my := ebiten.CursorPosition()
	if mx >= 0 && mx <= cityWindowWidth && my >= 0 && my <= cityWindowHeight {
		w.hoverIndex = (my + w.scrollY - 50) / 40
	} else {
		w.hoverIndex = -1
	}
}

func (w *CityWindow) Draw(screen *ebiten.Image) {
	if !w.open {
		return
	}

	ebitenutil.DrawRect(screen, 0, 0, cityWindowWidth, cityWindowHeight, color.RGBA{30, 30, 40, 230})

	title := "Список городов"
	bounds := text.BoundString(w.game.font, title)
	text.Draw(screen, title, w.game.font, cityWindowWidth/2-bounds.Max.X/2, 30, color.White)

	maxVisible := (cityWindowHeight - 80) / 40
	startIndex := w.scrollY / 40
	if startIndex < 0 {
		startIndex = 0
	}
	endIndex := startIndex + maxVisible + 1
	if endIndex > len(w.game.cityList) {
		endIndex = len(w.game.cityList)
	}

	for i := startIndex; i < endIndex; i++ {
		city := w.game.cityList[i]
		yPos := 50 + i*40 - w.scrollY

		if i == w.hoverIndex {
			ebitenutil.DrawRect(screen, 10, float64(yPos-20), cityWindowWidth-20, 30, color.RGBA{70, 70, 90, 150})
		}

		text.Draw(screen, city.Name, w.game.font, 20, yPos, color.White)
		info := fmt.Sprintf("Население: %d | Размер: %d", city.Population, city.Size)
		text.Draw(screen, info, w.game.font, 20, yPos+15, color.RGBA{180, 180, 180, 255})
	}

	if w.selected != nil {
		info := fmt.Sprintf("Выбран: %s\nНаселение: %d\nКоординаты: (%d, %d)",
			w.selected.Name, w.selected.Population, w.selected.X, w.selected.Y)
		text.Draw(screen, info, w.game.font, 20, cityWindowHeight-80, color.RGBA{255, 255, 0, 255})
	}

	if len(w.game.cityList) > maxVisible {
		scrollHeight := cityWindowHeight * maxVisible / len(w.game.cityList)
		scrollPos := cityWindowHeight * w.scrollY / (len(w.game.cityList) * 40)
		ebitenutil.DrawRect(screen, cityWindowWidth-10, float64(scrollPos), 5, float64(scrollHeight), color.RGBA{150, 150, 150, 200})
	}

}
