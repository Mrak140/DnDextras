package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
	numSites     = 100
)

// Палитра цветов для цифр 1-10
var digitColors = []color.RGBA{
	{255, 0, 0, 255},     // 1: Красный
	{0, 255, 0, 255},     // 2: Зелёный
	{0, 0, 255, 255},     // 3: Синий
	{255, 255, 0, 255},   // 4: Жёлтый
	{255, 0, 255, 255},   // 5: Пурпурный
	{0, 255, 255, 255},   // 6: Бирюзовый
	{128, 0, 0, 255},     // 7: Тёмно-красный
	{0, 128, 0, 255},     // 8: Тёмно-зелёный
	{0, 0, 128, 255},     // 9: Тёмно-синий
	{128, 128, 128, 255}, // 10: Серый
}

type Game struct {
	sites []Point
}

type Point struct {
	x, y  float64
	digit int
}

func NewGame() *Game {
	g := &Game{}
	g.generateRandomSites(numSites)
	return g
}

func (g *Game) generateRandomSites(count int) {
	g.sites = make([]Point, count)
	for i := 0; i < count; i++ {
		g.sites[i] = Point{
			x:     rand.Float64() * screenWidth,
			y:     rand.Float64() * screenHeight,
			digit: rand.Intn(10) + 1,
		}
	}
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Отрисовка диаграммы Вороного
	g.drawVoronoi(screen)

	// Отрисовка цифр
	for _, site := range g.sites {
		text := strconv.Itoa(site.digit)
		textX := int(site.x) - 5*len(text)/2
		textY := int(site.y) + 5
		ebitenutil.DebugPrintAt(screen, text, textX, textY)
	}
}

func (g *Game) drawVoronoi(screen *ebiten.Image) {
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			minDist := math.MaxFloat64
			closestDigit := 1

			for _, site := range g.sites {
				dx := float64(x) - site.x
				dy := float64(y) - site.y
				dist := dx*dx + dy*dy

				if dist < minDist {
					minDist = dist
					closestDigit = site.digit
				}
			}

			// Берём цвет из палитры по цифре (closestDigit - 1, т.к. индексы с 0)
			screen.Set(x, y, digitColors[closestDigit-1])
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Voronoi Diagram: One Color per Digit")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
