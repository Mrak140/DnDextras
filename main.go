package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth   = 1920
	screenHeight  = 1080
	numSites      = 100
	borderWidth   = 2
	workers       = 8
	wfcIterations = 10 // Уменьшено для производительности
)

var (
	digitColors = []color.RGBA{
		{50, 200, 50, 255},   // 1: Зеленый
		{10, 10, 100, 255},   // 2: Темно-синий
		{100, 200, 255, 255}, // 3: Голубой
		{255, 255, 0, 255},   // 4: Желтый
		{255, 0, 255, 255},   // 5: Пурпурный
		{0, 255, 255, 255},   // 6: Бирюзовый
		{128, 0, 0, 255},     // 7: Темно-красный
		{0, 128, 0, 255},     // 8: Темно-зеленый
		{0, 0, 128, 255},     // 9: Темно-синий
		{255, 0, 0, 1},       // 10: Красный
	}
	black = color.RGBA{0, 0, 0, 255}
)

type Game struct {
	sites     []Point
	nearest   [][]int
	wfcGrid   [][]int
	pixels    []byte
	ready     bool
	calcMutex sync.Mutex
}

type Point struct {
	x, y  float64
	digit int
}

func NewGame() *Game {
	g := &Game{
		pixels: make([]byte, screenWidth*screenHeight*4),
	}
	g.generateInitialSites(numSites)
	go g.precompute()
	return g
}

func (g *Game) generateInitialSites(count int) {
	g.sites = make([]Point, 0, count)
	rand.Seed(time.Now().UnixNano())

	// Гарантируем базовое распределение
	for digit := 1; digit <= 10; digit++ {
		g.sites = append(g.sites, Point{
			x:     rand.Float64() * screenWidth,
			y:     rand.Float64() * screenHeight,
			digit: digit,
		})
	}

	// Добавляем больше точек для 1, 2, 3
	for i := 10; i < count; i++ {
		digit := rand.Intn(3) + 1 // 60% chance for 1-3
		if rand.Float32() > 0.6 {
			digit = rand.Intn(7) + 4
		}
		g.sites = append(g.sites, Point{
			x:     rand.Float64() * screenWidth,
			y:     rand.Float64() * screenHeight,
			digit: digit,
		})
	}
}

func (g *Game) precompute() {
	g.calcMutex.Lock()
	defer g.calcMutex.Unlock()

	// Этап 1: Быстрое вычисление Voronoi
	g.nearest = make([][]int, screenHeight)
	for y := range g.nearest {
		g.nearest[y] = make([]int, screenWidth)
	}

	// Параллельное вычисление
	var wg sync.WaitGroup
	rowsPerWorker := screenHeight / workers

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startY := workerID * rowsPerWorker
			endY := (workerID + 1) * rowsPerWorker
			if workerID == workers-1 {
				endY = screenHeight
			}

			for y := startY; y < endY; y++ {
				for x := 0; x < screenWidth; x++ {
					minDist := math.MaxFloat64
					closest := 0

					for i, site := range g.sites {
						dx := float64(x) - site.x
						dy := float64(y) - site.y
						dist := dx*dx + dy*dy
						if dist < minDist {
							minDist = dist
							closest = g.sites[i].digit
						}
					}
					g.nearest[y][x] = closest
				}
			}
		}(w)
	}
	wg.Wait()

	// Этап 2: Оптимизированный WFC
	g.applyOptimizedWFC()

	// Этап 3: Предварительная отрисовка
	g.renderToPixels()

	g.ready = true
}

func (g *Game) applyOptimizedWFC() {
	g.wfcGrid = make([][]int, screenHeight)
	for y := range g.wfcGrid {
		g.wfcGrid[y] = make([]int, screenWidth)
		copy(g.wfcGrid[y], g.nearest[y])
	}

	// Оптимизированные правила соседства
	allowedNeighbors := map[int]map[int]bool{
		1: {1: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true},
		2: {2: true, 3: true},
		3: {1: true, 2: true, 3: true},
	}

	for iter := 0; iter < wfcIterations; iter++ {
		for y := 1; y < screenHeight-1; y++ {
			for x := 1; x < screenWidth-1; x++ {
				current := g.wfcGrid[y][x]
				rules, hasRules := allowedNeighbors[current]

				if hasRules {
					// Проверяем только 4 соседа
					neighbors := [4]int{
						g.wfcGrid[y-1][x], g.wfcGrid[y+1][x],
						g.wfcGrid[y][x-1], g.wfcGrid[y][x+1],
					}

					for _, n := range neighbors {
						if !rules[n] {
							// Находим наиболее подходящую замену
							g.wfcGrid[y][x] = g.findBestReplacement(x, y)
							break
						}
					}
				}
			}
		}
	}
}

func (g *Game) findBestReplacement(x, y int) int {
	// Анализ окружения
	counts := make(map[int]int)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dy == 0 && dx == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && ny >= 0 && nx < screenWidth && ny < screenHeight {
				counts[g.wfcGrid[ny][nx]]++
			}
		}
	}

	// Выбираем наиболее подходящий вариант
	bestDigit := 3 // По умолчанию - универсальный 3
	maxCount := 0
	for digit, count := range counts {
		if count > maxCount {
			bestDigit = digit
			maxCount = count
		}
	}
	return bestDigit
}

func (g *Game) renderToPixels() {
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			idx := (y*screenWidth + x) * 4
			digit := g.wfcGrid[y][x]
			color := digitColors[digit-1]

			// Основной цвет
			g.pixels[idx] = color.R
			g.pixels[idx+1] = color.G
			g.pixels[idx+2] = color.B
			g.pixels[idx+3] = color.A

			// Границы
			if y > 0 && x > 0 && y < screenHeight-1 && x < screenWidth-1 {
				current := g.wfcGrid[y][x]
				if g.wfcGrid[y-1][x] != current || g.wfcGrid[y+1][x] != current ||
					g.wfcGrid[y][x-1] != current || g.wfcGrid[y][x+1] != current {
					g.pixels[idx] = 0
					g.pixels[idx+1] = 0
					g.pixels[idx+2] = 0
				}
			}
		}
	}
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.calcMutex.Lock()
	defer g.calcMutex.Unlock()

	if !g.ready {
		return
	}

	screen.ReplacePixels(g.pixels)

	// Отрисовка цифр поверх
	for _, site := range g.sites {
		text := strconv.Itoa(site.digit)
		textX := int(site.x) - 5*len(text)/2
		textY := int(site.y) + 5
		ebitenutil.DebugPrintAt(screen, text, textX, textY)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Optimized Voronoi-WFC")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
