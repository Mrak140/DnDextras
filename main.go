package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 800
	screenHeight = 600
	numSites     = 50 // Количество точек (сайтов)
)

type Game struct {
	sites []Point
}

type Point struct {
	x, y float64
	col  color.RGBA
}

func NewGame() *Game {
	g := &Game{}
	g.generateRandomSites(numSites)
	return g
}

// Генерация случайных точек
func (g *Game) generateRandomSites(count int) {
	g.sites = make([]Point, count)
	for i := 0; i < count; i++ {
		g.sites[i] = Point{
			x: rand.Float64() * screenWidth,
			y: rand.Float64() * screenHeight,
			col: color.RGBA{
				R: uint8(rand.Intn(200) + 55),
				G: uint8(rand.Intn(200) + 55),
				B: uint8(rand.Intn(200) + 55),
				A: 255,
			},
		}
	}
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Отрисовка диаграммы Вороного
	g.drawVoronoi(screen)

	// Отрисовка точек (сайтов)
	for _, site := range g.sites {
		ebitenutil.DrawCircle(screen, site.x, site.y, 4, color.Black)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Voronoi Diagram in Ebiten")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

// Наивный алгоритм Вороного (перебор всех пикселей)
func (g *Game) drawVoronoi(screen *ebiten.Image) {
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			// Находим ближайшую точку
			minDist := math.MaxFloat64
			var closestSite Point

			for _, site := range g.sites {
				dx := float64(x) - site.x
				dy := float64(y) - site.y
				dist := dx*dx + dy*dy // Квадрат расстояния (оптимизация)

				if dist < minDist {
					minDist = dist
					closestSite = site
				}
			}

			// Закрашиваем пиксель цветом ближайшей точки
			screen.Set(x, y, closestSite.col)
		}
	}
}
