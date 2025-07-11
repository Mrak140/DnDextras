package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth     = 1920
	screenHeight    = 1080
	initialCols     = 6
	initialRows     = 6
	finalCols       = 24
	finalRows       = 24
	biomeCols       = 96 // 24*4
	biomeRows       = 96
	biomeCellWidth  = screenWidth / biomeCols
	biomeCellHeight = screenHeight / biomeRows
)

type Humidity int

const (
	Arid Humidity = iota
	Dry
	Moist
	Wet
)

type Biome int

const (
	Water Biome = iota
	DeepWater
	Tundra
	Desert
	Volcano
	Forest
	Meadow
	Jungle
	IceMountains
)

type Game struct {
	initialGrid  [][]bool
	finalGrid    [][]bool
	temperature  *TemperatureMap
	humidityGrid [][]Humidity
	biomeGrid    [][]Biome
	showInitial  bool
	seed         int64
}

func NewGame() *Game {
	g := &Game{
		initialGrid: make([][]bool, initialRows),
		seed:        time.Now().UnixNano(),
	}
	for i := range g.initialGrid {
		g.initialGrid[i] = make([]bool, initialCols)
	}
	g.Regenerate()
	return g
}

func (g *Game) distanceToWater(x, y int) float64 {
	minDist := math.MaxFloat64

	for ny := 0; ny < finalRows; ny++ {
		for nx := 0; nx < finalCols; nx++ {
			if !g.finalGrid[ny][nx] {
				dist := math.Sqrt(float64((nx-x)*(nx-x) + (ny-y)*(ny-y)))
				if dist < minDist {
					minDist = dist
				}
			}
		}
	}

	maxPossibleDist := math.Sqrt(float64(finalCols*finalCols + finalRows*finalRows))
	return minDist / maxPossibleDist
}

func (g *Game) generateHumidity() {
	g.humidityGrid = make([][]Humidity, finalRows)
	for y := 0; y < finalRows; y++ {
		g.humidityGrid[y] = make([]Humidity, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				continue
			}
			dist := g.distanceToWater(x, y)
			switch {
			case dist < 0.2:
				g.humidityGrid[y][x] = Wet
			case dist < 0.4:
				g.humidityGrid[y][x] = Moist
			case dist < 0.7:
				g.humidityGrid[y][x] = Dry
			default:
				g.humidityGrid[y][x] = Arid
			}
		}
	}
}

func (g *Game) generateBiomes() {
	baseBiomes := make([][]Biome, finalRows)
	for y := 0; y < finalRows; y++ {
		baseBiomes[y] = make([]Biome, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				baseBiomes[y][x] = Water
				continue
			}
			baseBiomes[y][x] = g.determineBiome(x, y)
		}
	}

	g.biomeGrid = make([][]Biome, biomeRows)
	for y := 0; y < biomeRows; y++ {
		g.biomeGrid[y] = make([]Biome, biomeCols)
		for x := 0; x < biomeCols; x++ {
			baseX, baseY := x/4, y/4
			if baseX >= finalCols || baseY >= finalRows {
				continue
			}
			g.biomeGrid[y][x] = baseBiomes[baseY][baseX]
		}
	}

	for i := 0; i < 3; i++ {
		g.smoothWaterEdges()
	}

	for i := 0; i < 5; i++ {
		g.smoothBiomesEnhanced()
	}
}

func (g *Game) smoothWaterEdges() {
	newGrid := make([][]Biome, biomeRows)
	for y := 0; y < biomeRows; y++ {
		newGrid[y] = make([]Biome, biomeCols)
		for x := 0; x < biomeCols; x++ {
			current := g.biomeGrid[y][x]
			waterNeighbors := g.countWaterNeighborsInBiomeGrid(x, y)

			// Новые правила для более естественных границ
			switch {

			case current != Water:
				// Более плавное проникновение воды в сушу
				if waterNeighbors >= 6 {
					newGrid[y][x] = Water
				} else if waterNeighbors >= 4 {
					if rand.Float32() < 0.3 {
						newGrid[y][x] = Water
					} else {
						newGrid[y][x] = current
					}
				} else {
					newGrid[y][x] = current
				}
			}
		}
	}
	g.biomeGrid = newGrid
}

func (g *Game) smoothWaterBorders() {
	newGrid := make([][]bool, finalRows)
	for y := 0; y < finalRows; y++ {
		newGrid[y] = make([]bool, finalCols)
		for x := 0; x < finalCols; x++ {
			waterNeighbors := g.countWaterNeighbors(x, y)
			current := g.finalGrid[y][x]

			// Правила сглаживания:
			if !current { // Если это вода
				if waterNeighbors == 8 { // Полностью окружена водой
					newGrid[y][x] = false // Остается водой (глубокую воду обработаем позже)
				} else if waterNeighbors >= 5 { // Больше воды вокруг
					newGrid[y][x] = false
				} else {
					// 50% шанс превратиться в землю, если соседствует с землей
					newGrid[y][x] = rand.Float32() < 0.5
				}
			} else {
				newGrid[y][x] = current
			}
		}
	}
	g.finalGrid = newGrid
}

func (g *Game) markDeepWater() {
	for y := 0; y < finalRows; y++ {
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] { // Если это вода
				waterNeighbors := g.countWaterNeighbors(x, y)
				if waterNeighbors == 8 { // Полностью окружена водой
					// Преобразуем координаты в biomeGrid
					biomeX, biomeY := x/4, y/4
					if biomeX < finalCols && biomeY < finalRows {
						g.biomeGrid[biomeY][biomeX] = DeepWater
					}
				}
			}
		}
	}
}

func (g *Game) countWaterNeighbors(x, y int) int {
	count := 0
	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx == x && ny == y {
				continue
			}
			if nx >= 0 && nx < finalCols && ny >= 0 && ny < finalRows {
				if !g.finalGrid[ny][nx] {
					count++
				}
			}
		}
	}
	return count
}

func (g *Game) smoothBiomesEnhanced() {
	newGrid := make([][]Biome, biomeRows)
	for y := 0; y < biomeRows; y++ {
		newGrid[y] = make([]Biome, biomeCols)
		for x := 0; x < biomeCols; x++ {
			current := g.biomeGrid[y][x]
			neighbors := g.getExtendedBiomeNeighbors(x, y)

			// Учитываем больше соседей
			dominantBiome := current
			maxCount := 0
			for biome, count := range neighbors {
				if count > maxCount {
					maxCount = count
					dominantBiome = biome
				}
			}

			// Более плавные правила перехода
			switch {
			case maxCount >= 8:
				newGrid[y][x] = dominantBiome
			case maxCount >= 5:
				if rand.Float32() < 0.8 {
					newGrid[y][x] = dominantBiome
				} else {
					newGrid[y][x] = current
				}
			default:
				newGrid[y][x] = current
			}
		}
	}
	g.biomeGrid = newGrid
}

func (g *Game) getExtendedBiomeNeighbors(x, y int) map[Biome]int {
	neighbors := make(map[Biome]int)
	// Проверяем 8 соседей вместо 4
	for ny := y - 2; ny <= y+2; ny++ {
		for nx := x - 2; nx <= x+2; nx++ {
			if nx == x && ny == y {
				continue // Пропускаем текущую клетку
			}
			if nx >= 0 && nx < biomeCols && ny >= 0 && ny < biomeRows && g.isLandInBaseGrid(nx, ny) {
				neighbors[g.biomeGrid[ny][nx]]++
			}
		}
	}
	return neighbors
}

func (g *Game) countWaterNeighborsInBiomeGrid(x, y int) int {
	count := 0
	// Проверяем 24 соседа (5x5 область) для более плавных переходов
	for ny := y - 2; ny <= y+2; ny++ {
		for nx := x - 2; nx <= x+2; nx++ {
			if nx == x && ny == y {
				continue
			}
			if nx >= 0 && nx < biomeCols && ny >= 0 && ny < biomeRows {
				if g.biomeGrid[ny][nx] == Water {
					// Вес зависит от расстояния
					dist := math.Sqrt(float64((nx-x)*(nx-x) + (ny-y)*(ny-y)))
					weight := 1.0 / (dist + 1)
					count += int(weight * 3)
				}
			}
		}
	}
	return count
}

func (g *Game) isLandInBaseGrid(x, y int) bool {
	baseX, baseY := x/4, y/4 // Изменили делитель с 2 на 4
	return baseX < finalCols && baseY < finalRows && g.finalGrid[baseY][baseX]
}

func (g *Game) determineBiome(x, y int) Biome {
	temp := g.temperature.GetTemperature(x, y)
	humid := g.humidityGrid[y][x]

	switch temp {
	case Frozen:
		// Ледяные горы для замороженных зон
		return IceMountains
	case Cold:
		// Тундра для холодных зон
		return Tundra
	case Cool:
		// Для прохладных зон выбираем между лесом, джунглями и лугами в зависимости от влажности
		switch humid {
		case Wet:
			return Jungle
		case Moist:
			return Forest
		default:
			return Meadow
		}
	case Warm:
		// Пустыни для теплых зон
		return Desert
	case Hot:
		// Вулканы для самых горячих зон
		return Volcano
	default:
		// По умолчанию возвращаем луг
		return Meadow
	}
}

func (g *Game) Regenerate() {
	rand.Seed(g.seed)

	for y := 0; y < initialRows; y++ {
		for x := 0; x < initialCols; x++ {
			g.initialGrid[y][x] = rand.Float32() < 0.35
		}
	}

	g.upscaleAndSmooth()
	g.temperature = NewTemperatureMap(g.seed, finalCols, finalRows)
	g.generateHumidity()
	g.generateBiomes()
	g.markDeepWater()
}

func (g *Game) upscaleAndSmooth() {
	intermediateCols := finalCols / 2
	intermediateRows := finalRows / 2
	intermediateGrid := make([][]bool, intermediateRows)

	for y := 0; y < intermediateRows; y++ {
		intermediateGrid[y] = make([]bool, intermediateCols)
		for x := 0; x < intermediateCols; x++ {
			intermediateGrid[y][x] = g.initialGrid[y/2][x/2]
		}
	}

	g.finalGrid = make([][]bool, finalRows)
	for y := 0; y < finalRows; y++ {
		g.finalGrid[y] = make([]bool, finalCols)
		for x := 0; x < finalCols; x++ {
			baseValue := intermediateGrid[y/2][x/2]
			if rand.Float32() < 0.15 {
				baseValue = !baseValue
			}
			g.finalGrid[y][x] = baseValue
		}
	}

	for i := 0; i < 5; i++ {
		g.applyCellularAutomaton()
	}

	g.smoothWaterBorders()
}

func (g *Game) applyCellularAutomaton() {
	newGrid := make([][]bool, finalRows)
	for y := 0; y < finalRows; y++ {
		newGrid[y] = make([]bool, finalCols)
		for x := 0; x < finalCols; x++ {
			count := g.countLandNeighbors(x, y)
			// Более мягкие правила для сглаживания
			if g.finalGrid[y][x] {
				newGrid[y][x] = count >= 3 // Было 4
			} else {
				newGrid[y][x] = count >= 5
			}
		}
	}
	g.finalGrid = newGrid
}

func (g *Game) countLandNeighbors(x, y int) int {
	count := 0
	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx == x && ny == y {
				continue
			}
			if nx >= 0 && nx < finalCols && ny >= 0 && ny < finalRows && g.finalGrid[ny][nx] {
				count++
			}
		}
	}
	return count
}

func (g *Game) Update() error {
	// Переключение режима отображения по пробелу
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.showInitial = !g.showInitial
	}

	// Регенерация карты по клику мыши
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.seed = time.Now().UnixNano()
		g.Regenerate()
	}

	return nil
}

func (g *Game) drawDebugInfo(screen *ebiten.Image) {
	debugText := fmt.Sprintf(
		"Seed: %d\n"+
			"Biomes:\n"+
			"Tundra (белый)\n"+
			"Desert (песок)\n"+
			"Volcano (красный)\n"+
			"Forest (зеленый)\n"+
			"Meadow (салатовый)\n"+
			"Jungle (темно-зеленый)\n"+
			"IceMountains (голубой)\n"+
			"WASD: перемещение\n"+
			"R: регенерация",
		g.seed,
	)
	ebitenutil.DebugPrint(screen, debugText)
}

func (g *Game) Draw(screen *ebiten.Image) {
	for y := 0; y < biomeRows; y++ {
		for x := 0; x < biomeCols; x++ {
			var biomeColor color.RGBA
			switch g.biomeGrid[y][x] {
			case Water:
				biomeColor = color.RGBA{70, 130, 180, 255} // Стандартная вода
			case DeepWater:
				biomeColor = color.RGBA{25, 55, 100, 255} // Темная глубокая вода
			case Tundra:
				biomeColor = color.RGBA{245, 245, 255, 255}
			case Desert:
				biomeColor = color.RGBA{210, 185, 100, 255}
			case Volcano:
				biomeColor = color.RGBA{200, 50, 30, 255} // Ярко-красный для вулканов
			case Forest:
				biomeColor = color.RGBA{50, 120, 50, 255}
			case Meadow:
				biomeColor = color.RGBA{120, 200, 80, 255}
			case Jungle:
				biomeColor = color.RGBA{30, 90, 30, 255}
			case IceMountains:
				biomeColor = color.RGBA{180, 220, 255, 255} // Голубовато-белый для ледяных гор
			default:
				biomeColor = color.RGBA{120, 120, 120, 255}
			}

			// Рисуем клетку биома
			ebitenutil.DrawRect(
				screen,
				float64(x*biomeCellWidth),
				float64(y*biomeCellHeight),
				biomeCellWidth,
				biomeCellHeight,
				biomeColor,
			)
		}
	}

	// Рисуем отладочную информацию
	g.drawDebugInfo(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Procedural Biome Map")

	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
