package main

import (
	"fmt"
	"image"
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
	initialWidth    = 320
	initialHeight   = 180
	biomeCols       = 96 // 24*4 (так как масштабируем в 4 раза)
	biomeRows       = 96
	biomeCellWidth  = screenWidth / biomeCols
	biomeCellHeight = screenHeight / biomeRows
	heightLevels    = 5 // Количество уровней высоты
	finalWidth      = screenWidth / finalCols
	finalHeight     = screenHeight / finalRows
)

// Типы влажности
type Humidity int

const (
	Arid Humidity = iota
	Dry
	Moist
	Wet
)

// Реализация шума Перлина (исправленная)
type perlinNoise struct {
	seed    int64
	octaves int
	pers    float64
	lac     float64
}

func perlinNoiseGenerator(octaves int, persistence, lacunarity float64, seed int64) *perlinNoise {
	return &perlinNoise{
		seed:    seed,
		octaves: octaves,
		pers:    persistence,
		lac:     lacunarity,
	}
}

func (p *perlinNoise) Noise2D(x, y float64) float64 {
	var total float64
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < p.octaves; i++ {
		total += p.interpolatedNoise(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= p.pers
		frequency *= p.lac
	}

	return total / maxValue
}

func (p *perlinNoise) interpolatedNoise(x, y float64) float64 {
	ix := int(x)
	iy := int(y)
	fx := x - float64(ix)
	fy := y - float64(iy)

	v1 := p.smoothNoise(ix, iy)
	v2 := p.smoothNoise(ix+1, iy)
	v3 := p.smoothNoise(ix, iy+1)
	v4 := p.smoothNoise(ix+1, iy+1)

	i1 := p.interpolate(v1, v2, fx)
	i2 := p.interpolate(v3, v4, fx)

	return p.interpolate(i1, i2, fy)
}

func (p *perlinNoise) interpolate(a, b, x float64) float64 {
	ft := x * math.Pi
	f := (1 - math.Cos(ft)) * 0.5
	return a*(1-f) + b*f
}

func (p *perlinNoise) smoothNoise(x, y int) float64 {
	corners := (p.noise(x-1, y-1) + p.noise(x+1, y-1) + p.noise(x-1, y+1) + p.noise(x+1, y+1)) / 16
	sides := (p.noise(x-1, y) + p.noise(x+1, y) + p.noise(x, y-1) + p.noise(x, y+1)) / 8
	center := p.noise(x, y) / 4
	return corners + sides + center
}

func (p *perlinNoise) noise(x, y int) float64 {
	n := x + y*57
	n = (n << 13) ^ n
	// Исправленная формула для генерации шума
	return 1.0 - float64((n*(n*n*15731+789221)+int(p.seed))&0x7fffffff)/1073741824.0
}

type Height int

const (
	WaterLevel Height = iota
	Lowland
	Hills
	Mountains
	HighMountains
)

type Game struct {
	initialGrid  [][]bool
	finalGrid    [][]bool
	tempGrid     [][]Temperature
	humidityGrid [][]Humidity
	biomeGrid    [][]Biome
	showInitial  bool
	seed         int64
	heightGrid   [][]Height
}

type Temperature int

const (
	Warm Temperature = iota
	Hot
	Cold
	Frozen
	Cool
	Snow
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

func (g *Game) generateHeightMap() {
	g.heightGrid = make([][]Height, finalRows)
	noise := perlinNoiseGenerator(3, 0.5, 2.0, g.seed) // Измененные параметры шума

	for y := 0; y < finalRows; y++ {
		g.heightGrid[y] = make([]Height, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				g.heightGrid[y][x] = WaterLevel
				continue
			}

			dist := g.distanceToWater(x, y)
			// Добавляем больше шума для естественного рельефа
			pnoise := (noise.Noise2D(float64(x)/10, float64(y)/10) + 1) / 2
			heightValue := (dist*0.6 + pnoise*0.4) * float64(heightLevels-1)

			// Немного случайности
			heightValue += rand.Float64() * 0.3

			g.heightGrid[y][x] = Height(math.Min(float64(heightLevels-1), heightValue))
		}
	}
}

func (g *Game) distanceToWater(x, y int) float64 {
	minDist := math.MaxFloat64

	// Проверяем все клетки карты, а не только соседей
	for ny := 0; ny < finalRows; ny++ {
		for nx := 0; nx < finalCols; nx++ {
			if !g.finalGrid[ny][nx] { // Если это вода
				dist := math.Sqrt(float64((nx-x)*(nx-x) + (ny-y)*(ny-y)))
				if dist < minDist {
					minDist = dist
				}
			}
		}
	}

	// Нормализуем расстояние
	maxPossibleDist := math.Sqrt(float64(finalCols*finalCols + finalRows*finalRows))
	return minDist / maxPossibleDist
}

func (g *Game) adjustTemperatureWithHeight() {
	for y := 0; y < finalRows; y++ {
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				continue
			}

			// Понижаем температуру с высотой
			heightEffect := float64(g.heightGrid[y][x]) * 0.15 // 15% на уровень
			r := rand.Float64() - heightEffect

			switch {
			case r < 0.60:
				g.tempGrid[y][x] = Warm
			case r < 0.80:
				g.tempGrid[y][x] = Cold
			default:
				g.tempGrid[y][x] = Frozen
			}
		}
	}
}

func (g *Game) generateHumidity() {
	g.humidityGrid = make([][]Humidity, finalRows)
	for y := 0; y < finalRows; y++ {
		g.humidityGrid[y] = make([]Humidity, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				continue
			}

			waterInfluence := 0.0
			// Учитываем большее расстояние до воды
			for ny := 0; ny < finalRows; ny++ {
				for nx := 0; nx < finalCols; nx++ {
					if !g.finalGrid[ny][nx] {
						dist := math.Sqrt(float64((nx-x)*(nx-x) + (ny-y)*(ny-y)))
						if dist < 8 { // Увеличили радиус влияния воды
							// Квадратичное затухание влияния
							waterInfluence += 1.0 / (dist*dist + 1)
						}
					}
				}
			}

			// Сильнее зависимость от высоты
			heightEffect := math.Pow(float64(g.heightGrid[y][x])/float64(heightLevels-1), 1.5)
			waterInfluence *= (1 - heightEffect*0.8)

			// Добавляем немного случайности
			waterInfluence += rand.Float64()*0.2 - 0.1

			switch {
			case waterInfluence >= 2.0:
				g.humidityGrid[y][x] = Wet
			case waterInfluence >= 1.0:
				g.humidityGrid[y][x] = Moist
			case waterInfluence >= 0.3:
				g.humidityGrid[y][x] = Dry
			default:
				g.humidityGrid[y][x] = Arid
			}
		}
	}
}

func (g *Game) generateBiomes() {
	// 1. Создаем базовую сетку 24x24
	baseBiomes := make([][]Biome, finalRows)
	for y := 0; y < finalRows; y++ {
		baseBiomes[y] = make([]Biome, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				baseBiomes[y][x] = Water
				continue
			}
			temp := g.tempGrid[y][x]
			humid := g.humidityGrid[y][x]
			height := g.heightGrid[y][x]
			baseBiomes[y][x] = g.determineBiome(temp, humid, height, x, y)
		}
	}

	// 2. Увеличиваем в 4 раза до 96x96
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

	// 3. Специальное сглаживание границ воды (3 прохода)
	for i := 0; i < 3; i++ {
		g.smoothWaterEdges()
	}

	// 4. Основное сглаживание биомов
	for i := 0; i < 5; i++ {
		g.smoothBiomesEnhanced()
	}

	// 5. Финальное сглаживание с шумом
	g.applyCoastalNoise()
}

func (g *Game) smoothWaterEdges() {
	newGrid := make([][]Biome, biomeRows)
	for y := 0; y < biomeRows; y++ {
		newGrid[y] = make([]Biome, biomeCols)
		for x := 0; x < biomeCols; x++ {
			current := g.biomeGrid[y][x]
			waterNeighbors := g.countWaterNeighborsInBiomeGrid(x, y)
			landNeighbors := 8 - waterNeighbors

			// Новые правила для более естественных границ
			switch {
			case current == Water:
				// Плавный переход от воды к суше
				if landNeighbors >= 5 {
					// Случайный выбор прибрежного биома для разнообразия
					if rand.Float32() < 0.7 {
						newGrid[y][x] = g.getCoastalBiome(x, y)
					} else {
						newGrid[y][x] = Water
					}
				} else if landNeighbors >= 3 {
					// 50% шанс превращения в прибрежный биом
					if rand.Float32() < 0.5 {
						newGrid[y][x] = g.getCoastalBiome(x, y)
					} else {
						newGrid[y][x] = Water
					}
				} else {
					newGrid[y][x] = Water
				}

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

func (g *Game) getCoastalBiome(x, y int) Biome {
	baseX, baseY := x/4, y/4
	if baseX >= 0 && baseX < finalCols && baseY >= 0 && baseY < finalRows {
		temp := g.tempGrid[baseY][baseX]

		// Выбираем прибрежный биом в зависимости от температуры
		switch temp {
		case Frozen:
			return Tundra
		case Hot:
			if rand.Float32() < 0.3 {
				return Desert
			}
			return Meadow
		default:
			return Meadow
		}
	}
	return Meadow
}

func (g *Game) applyCoastalNoise() {
	noise := perlinNoiseGenerator(4, 0.5, 2.0, g.seed+1) // Отдельный seed для шума воды

	for y := 1; y < biomeRows-1; y++ {
		for x := 1; x < biomeCols-1; x++ {
			if g.biomeGrid[y][x] == Water {
				continue
			}

			waterNeighbors := g.countWaterNeighborsInBiomeGrid(x, y)
			if waterNeighbors == 0 {
				continue
			}

			// Используем шум Перлина для плавных изменений
			n := (noise.Noise2D(float64(x)/10, float64(y)/10) + 1) / 2 // Нормализуем от 0 до 1

			// Чем больше соседей-воды, тем выше вероятность изменения
			probability := float32(waterNeighbors) / 8 * 0.5
			probability += float32(n) * 0.3

			if rand.Float32() < probability {
				if g.biomeGrid[y][x] != Water {
					g.biomeGrid[y][x] = g.getCoastalBiome(x, y)
				}
			}
		}
	}
}

func (g *Game) smoothBiomesEnhanced() {
	newGrid := make([][]Biome, biomeRows)
	for y := 0; y < biomeRows; y++ {
		newGrid[y] = make([]Biome, biomeCols)
		for x := 0; x < biomeCols; x++ {
			current := g.biomeGrid[y][x]
			neighbors := g.getExtendedBiomeNeighbors(x, y)
			waterNeighbors := g.countWaterNeighborsInBiomeGrid(x, y)

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
			case waterNeighbors > 0 && current != Water:
				if rand.Float32() < 0.9-float32(waterNeighbors)/10 {
					newGrid[y][x] = g.getCoastalBiome(x, y)
				} else {
					newGrid[y][x] = current
				}
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

func (g *Game) determineBiome(temp Temperature, humid Humidity, height Height, x, y int) Biome {
	switch temp {
	case Frozen:
		if height >= Mountains {
			// Проверяем, нет ли воды рядом
			if x >= 0 && x < finalCols && y >= 0 && y < finalRows {
				if g.hasWaterNeighbors(x, y) {
					return Tundra // У воды вместо гор делаем тундру
				}
			}
			return IceMountains
		}
		return Tundra

	case Hot:
		if height >= Hills {
			if humid == Wet {
				return Volcano
			}
			return Forest // Горы в жарком климате
		}
		if humid == Arid {
			return Desert
		}
		return Meadow

	case Warm:
		switch height {
		case HighMountains:
			return IceMountains
		case Mountains:
			if humid == Wet {
				return Jungle
			}
			return Forest
		default:
			if humid == Wet {
				return Jungle
			}
			if humid == Moist {
				return Meadow
			}
			return Forest
		}
	case Cold:
		return Tundra
	default:
		return Meadow
	}
}

func (g *Game) hasWaterNeighbors(x, y int) bool {
	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx == x && ny == y {
				continue
			}
			if nx >= 0 && nx < finalCols && ny >= 0 && ny < finalRows {
				if !g.finalGrid[ny][nx] { // Если это вода
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) Regenerate() {
	rand.Seed(g.seed)

	// 1. Генерация базовой сетки 6x6
	for y := 0; y < initialRows; y++ {
		g.initialGrid[y] = make([]bool, initialCols)
		for x := 0; x < initialCols; x++ {
			g.initialGrid[y][x] = rand.Float32() < 0.35
		}
	}

	// 2. Масштабирование и сглаживание
	g.upscaleAndSmooth()

	// 3. Генерация карты высот
	g.generateHeightMap()

	// 4. Генерация температур
	g.assignTemperatures()

	// 5. Корректировка температуры
	g.adjustTemperatureWithHeight()

	// 6. Генерация влажности
	g.generateHumidity()

	// 7. Генерация биомов (инициализирует biomeGrid)
	g.generateBiomes()

	// 8. Пометка глубокой воды (теперь biomeGrid инициализирован)
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

// 3-й этап: распределение температур (4:1:1)

func (g *Game) assignTemperatures() {
	g.tempGrid = make([][]Temperature, finalRows)
	for y := 0; y < finalRows; y++ {
		g.tempGrid[y] = make([]Temperature, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				continue
			}

			// Сильнее зависимость от широты
			latitudeEffect := math.Pow(float64(y)/float64(finalRows), 2)
			heightEffect := float64(g.heightGrid[y][x]) * 0.25

			// Добавляем шум для плавности
			noise := rand.Float64() * 0.2
			r := noise + latitudeEffect*0.7 - heightEffect*0.5

			switch {
			case r < 0.15:
				g.tempGrid[y][x] = Frozen
			case r < 0.35:
				g.tempGrid[y][x] = Cold
			case r < 0.80:
				g.tempGrid[y][x] = Warm
			default:
				g.tempGrid[y][x] = Hot
			}
		}
	}

	// Больше итераций сглаживания
	for i := 0; i < 10; i++ {
		g.applyTemperatureAutomaton()
	}
}

func (g *Game) applyTemperatureAutomaton() {
	newTempGrid := make([][]Temperature, finalRows)
	for y := 0; y < finalRows; y++ {
		newTempGrid[y] = make([]Temperature, finalCols)
		for x := 0; x < finalCols; x++ {
			if !g.finalGrid[y][x] {
				continue
			}

			neighbors := g.getTemperatureNeighbors(x, y, g.tempGrid)

			// Система приоритетов (от высшего к низшему)
			switch {
			// 1. Правило для горячих зон (высший приоритет)
			case neighbors[Hot] > 0 && (neighbors[Cold] > 0 || neighbors[Frozen] > 0):
				newTempGrid[y][x] = Warm

			// 2. Правило для снега
			case neighbors[Frozen] >= 2:
				newTempGrid[y][x] = Snow

			// 3. Правило для горячих зон (классическое)
			case neighbors[Warm] >= 6:
				newTempGrid[y][x] = Hot

			// 4. Правило для льда
			case neighbors[Cold] >= 2:
				newTempGrid[y][x] = Frozen

			// 5. Правило для прохладных зон
			case neighbors[Warm] > 0 && neighbors[Cold] > 0:
				newTempGrid[y][x] = Cool

			// Сохраняем текущую температуру
			default:
				newTempGrid[y][x] = g.tempGrid[y][x]
			}
		}
	}
	g.tempGrid = newTempGrid
}

// Возвращает карту соседей по температуре

func (g *Game) getTemperatureNeighbors(x, y int, grid [][]Temperature) map[Temperature]int {
	neighbors := map[Temperature]int{
		Warm:   0,
		Hot:    0,
		Cold:   0,
		Frozen: 0,
		Cool:   0,
		Snow:   0,
	}

	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx == x && ny == y {
				continue
			}
			if nx >= 0 && nx < finalCols && ny >= 0 && ny < finalRows && g.finalGrid[ny][nx] {
				temp := grid[ny][nx]
				neighbors[temp]++
			}
		}
	}
	return neighbors
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

var emptySubImage = ebiten.NewImage(1, 1).SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image)

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
				biomeColor = color.RGBA{200, 50, 30, 255}
			case Forest:
				biomeColor = color.RGBA{50, 120, 50, 255}
			case Meadow:
				biomeColor = color.RGBA{120, 200, 80, 255}
			case Jungle:
				biomeColor = color.RGBA{30, 90, 30, 255}
			case IceMountains:
				biomeColor = color.RGBA{180, 220, 255, 255}
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
