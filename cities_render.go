package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func (g *Game) drawCities(screen *ebiten.Image) {
	if g.font == nil || len(g.cityList) == 0 {
		return
	}

	for _, city := range g.cityList {
		screenX := city.X*cellSize - g.cameraX
		screenY := city.Y*cellSize - g.cameraY

		// Простая проверка видимости
		if screenX < -cellSize || screenX > screenWidth ||
			screenY < -cellSize || screenY > screenHeight {
			continue
		}

		// Рисуем город как квадрат на карте
		citySize := cellSize * (city.Size + 1)
		ebitenutil.DrawRect(
			screen,
			float64(screenX)-float64(citySize)/2+float64(cellSize)/2,
			float64(screenY)-float64(citySize)/2+float64(cellSize)/2,
			float64(citySize),
			float64(citySize),
			color.RGBA{200, 100, 50, 255},
		)

		// Для городов размером больше 1 рисуем название
		if city.Size > 1 {
			text.Draw(
				screen,
				city.Name,
				g.font,
				screenX-len(city.Name)*3,
				screenY-int(float64(citySize)/2)-5,
				color.White,
			)
		}
	}

	// Подсветка города под курсором
	if g.hoverCity != nil {
		g.drawCityHighlight(screen, g.hoverCity)
	}
}

func (g *Game) drawCityHighlight(screen *ebiten.Image, city *City) {
	screenX := city.X*cellSize - g.cameraX
	screenY := city.Y*cellSize - g.cameraY
	citySize := cellSize * (city.Size + 1)

	// Рисуем рамку выделения
	ebitenutil.DrawRect(
		screen,
		float64(screenX)-float64(citySize)/2+float64(cellSize)/2,
		float64(screenY)-float64(citySize)/2+float64(cellSize)/2,
		float64(citySize),
		2,
		color.RGBA{255, 255, 0, 255},
	)
	ebitenutil.DrawRect(
		screen,
		float64(screenX)-float64(citySize)/2+float64(cellSize)/2,
		float64(screenY)+float64(citySize)/2+float64(cellSize)/2-2,
		float64(citySize),
		2,
		color.RGBA{255, 255, 0, 255},
	)
	ebitenutil.DrawRect(
		screen,
		float64(screenX)-float64(citySize)/2+float64(cellSize)/2,
		float64(screenY)-float64(citySize)/2+float64(cellSize)/2,
		2,
		float64(citySize),
		color.RGBA{255, 255, 0, 255},
	)
	ebitenutil.DrawRect(
		screen,
		float64(screenX)+float64(citySize)/2+float64(cellSize)/2-2,
		float64(screenY)-float64(citySize)/2+float64(cellSize)/2,
		2,
		float64(citySize),
		color.RGBA{255, 255, 0, 255},
	)

	// Информация о городе
	info := fmt.Sprintf("%s\nНаселение: %d", city.Name, city.Population)
	text.Draw(
		screen,
		info,
		g.font,
		screenX-len(info)*3,
		screenY-int(float64(citySize)/2)-25,
		color.RGBA{255, 255, 0, 255},
	)
}
