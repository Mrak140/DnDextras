// wfc.go
package wfc

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"
)

const (
	TileEmpty = iota
	TileWater
	TileGrass
	TileResidential1x1
	TileResidential2x2
)

type Tile struct {
	ID          int
	Name        string
	Color       color.RGBA
	Rules       map[int][]int // Базовые правила соседства
	Size        int           // 1 для 1x1, 2 для 2x2 и т.д.
	Weight      float64
	ClusterSize int // Предпочтительный размер кластера
	MinDistance int // Минимальное расстояние до таких же зданий
}

type WFCOptions struct {
	Width        int
	Height       int
	Tiles        []Tile
	Seed         int64
	Iterations   int
	UrbanDensity float64 // 0.0 - 1.0
}

type WaveFunction struct {
	grid         [][][]bool
	collapsed    [][]int
	entropy      [][]float64
	tiles        []Tile
	rng          *rand.Rand
	options      WFCOptions
	urbanWeights []float64
}

func NewWFC(options WFCOptions) *WaveFunction {
	if options.Seed == 0 {
		options.Seed = time.Now().UnixNano()
	}

	wf := &WaveFunction{
		grid:      make([][][]bool, options.Height),
		collapsed: make([][]int, options.Height),
		entropy:   make([][]float64, options.Height),
		tiles:     options.Tiles,
		rng:       rand.New(rand.NewSource(options.Seed)),
		options:   options,
	}

	wf.initUrbanWeights()

	for y := 0; y < options.Height; y++ {
		wf.grid[y] = make([][]bool, options.Width)
		wf.collapsed[y] = make([]int, options.Width)
		wf.entropy[y] = make([]float64, options.Width)
		for x := 0; x < options.Width; x++ {
			wf.grid[y][x] = make([]bool, len(options.Tiles))
			for t := range options.Tiles {
				wf.grid[y][x][t] = true
			}
			wf.collapsed[y][x] = -1
			wf.entropy[y][x] = wf.calculateInitialEntropy(x, y)
		}
	}

	return wf
}

func (wf *WaveFunction) initUrbanWeights() {
	wf.urbanWeights = make([]float64, len(wf.tiles))
	for i, tile := range wf.tiles {
		baseWeight := tile.Weight

		switch {
		case tile.ID >= TileResidential1x1 && tile.ID <= TileResidential2x2:
			wf.urbanWeights[i] = baseWeight * wf.options.UrbanDensity
		default:
			wf.urbanWeights[i] = baseWeight * (1.0 - wf.options.UrbanDensity*0.5)
		}
	}
}

func (wf *WaveFunction) calculateInitialEntropy(x, y int) float64 {
	entropy := 0.0
	for t := range wf.tiles {
		if wf.tiles[t].Weight <= 0 {
			continue
		}
		entropy += wf.tiles[t].Weight
	}
	return entropy
}

func CreateOrganicCityTiles() []Tile {
	return []Tile{
		{
			ID:     TileGrass,
			Name:   "Grass",
			Color:  color.RGBA{100, 180, 60, 255}, // Яркая зелень
			Weight: 1.2,                           // Больший вес для травы
			Size:   1,
			Rules: map[int][]int{
				TileWater:          {TileGrass},
				TileResidential1x1: {TileGrass, TileEmpty},
				TileResidential2x2: {TileGrass, TileEmpty},
				TileGrass:          {TileWater, TileResidential1x1, TileResidential2x2, TileEmpty},
			},
		},
		{
			ID:     TileWater,
			Name:   "Water",
			Color:  color.RGBA{50, 120, 200, 255}, // Глубокий синий
			Weight: 0.4,
			Size:   1,
			Rules: map[int][]int{
				TileGrass: {TileWater},
				TileWater: {TileWater, TileGrass},
				TileEmpty: {TileWater},
			},
			ClusterSize: 5, // Крупные водоёмы
		},
		{
			ID:          TileResidential1x1,
			Name:        "Small House",
			Color:       color.RGBA{200, 160, 120, 255}, // Тёплый бежевый
			Weight:      0.7,
			Size:        1,
			ClusterSize: 4, // Крупные кластеры
			MinDistance: 2, // Минимальное расстояние между кластерами
			Rules: map[int][]int{
				TileGrass:          {TileResidential1x1},
				TileEmpty:          {TileResidential1x1},
				TileResidential1x1: {TileGrass, TileEmpty, TileResidential1x1},
				TileResidential2x2: {TileGrass, TileEmpty},
			},
		},
		{
			ID:          TileResidential2x2,
			Name:        "Large Building",
			Color:       color.RGBA{180, 140, 100, 255}, // Тёмный бежевый
			Weight:      0.3,
			Size:        2,
			ClusterSize: 3,
			MinDistance: 4,
			Rules: map[int][]int{
				TileGrass:          {TileResidential2x2},
				TileEmpty:          {TileResidential2x2},
				TileResidential1x1: {TileGrass, TileEmpty},
			},
		},
		{
			ID:     TileEmpty,
			Name:   "Empty",
			Color:  color.RGBA{140, 140, 140, 255}, // Серый
			Weight: 0.15,
			Size:   1,
			Rules: map[int][]int{
				TileGrass:          {TileEmpty},
				TileResidential1x1: {TileEmpty},
				TileResidential2x2: {TileEmpty},
				TileWater:          {TileEmpty},
			},
		},
	}
}

func (wf *WaveFunction) areCompatible(tileA, tileB int, dir [2]int) bool {
	// Специальные правила для воды
	if tileA == TileWater || tileB == TileWater {
		// Вода не должна быть рядом с жилыми зданиями
		if tileA == TileResidential1x1 || tileA == TileResidential2x2 ||
			tileB == TileResidential1x1 || tileB == TileResidential2x2 {
			return false
		}
	}
	// Если у тайла нет правил - разрешаем любое соседство
	if len(wf.tiles[tileA].Rules) == 0 || len(wf.tiles[tileB].Rules) == 0 {
		return true
	}

	// Проверяем правила для tileA
	if allowed, exists := wf.tiles[tileA].Rules[tileB]; exists {
		for _, allowedTile := range allowed {
			if allowedTile == tileB {
				return true
			}
		}
	}

	// Проверяем правила для tileB
	if allowed, exists := wf.tiles[tileB].Rules[tileA]; exists {
		for _, allowedTile := range allowed {
			if allowedTile == tileA {
				return true
			}
		}
	}

	return false
}

// Улучшенный метод collapseCell с обработкой ошибок
func (wf *WaveFunction) collapseCell(x, y int) bool {
	possible := wf.getPossibleTiles(x, y)
	if len(possible) == 0 {
		wf.setTile(x, y, TileGrass)
		return true
	}

	// Взвешенный случайный выбор
	totalWeight := 0.0
	weights := make([]float64, len(possible))
	for i, t := range possible {
		weights[i] = wf.tiles[t].Weight
		totalWeight += weights[i]
	}

	r := wf.rng.Float64() * totalWeight
	cumWeight := 0.0
	for i, t := range possible {
		cumWeight += weights[i]
		if r <= cumWeight {
			wf.setTile(x, y, t)
			return true
		}
	}

	// На всякий случай
	wf.setTile(x, y, possible[0])
	return true
}

func (wf *WaveFunction) canPlaceLargeTile(x, y, size int) bool {
	if x+size > wf.options.Width || y+size > wf.options.Height {
		return false
	}

	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			if wf.collapsed[y+dy][x+dx] != -1 {
				return false
			}
		}
	}
	return true
}

func (wf *WaveFunction) setTile(x, y, tileID int) {
	tile := wf.tiles[tileID]

	// Для больших зданий заполняем всю площадь
	for dy := 0; dy < tile.Size; dy++ {
		for dx := 0; dx < tile.Size; dx++ {
			if y+dy < wf.options.Height && x+dx < wf.options.Width {
				wf.collapsed[y+dy][x+dx] = tileID
				wf.entropy[y+dy][x+dx] = 0
				// Запрещаем все другие тайлы в этой области
				for t := range wf.grid[y+dy][x+dx] {
					wf.grid[y+dy][x+dx][t] = (t == tileID)
				}
			}
		}
	}
}

// isCollapsed checks if a cell is collapsed
func (wf *WaveFunction) isCollapsed(x, y int) bool {
	return wf.collapsed[y][x] != -1
}

func (wf *WaveFunction) propagate(x, y int) {
	stack := [][2]int{{x, y}}
	visited := make(map[[2]int]bool)

	for len(stack) > 0 {
		cx, cy := stack[len(stack)-1][0], stack[len(stack)-1][1]
		stack = stack[:len(stack)-1]

		if visited[[2]int{cx, cy}] {
			continue
		}
		visited[[2]int{cx, cy}] = true

		// Проверяем всех 4-х соседей
		for _, dir := range [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			nx, ny := cx+dir[0], cy+dir[1]

			if nx < 0 || ny < 0 || nx >= wf.options.Width || ny >= wf.options.Height {
				continue
			}

			if wf.collapsed[ny][nx] != -1 {
				continue
			}

			changed := false
			for t := 0; t < len(wf.tiles); t++ {
				if !wf.grid[ny][nx][t] {
					continue
				}

				compatible := false
				// Проверяем совместимость с текущей клеткой
				for ct := 0; ct < len(wf.tiles); ct++ {
					if !wf.grid[cy][cx][ct] {
						continue
					}
					if wf.areCompatible(ct, t, dir) {
						compatible = true
						break
					}
				}

				if !compatible {
					wf.grid[ny][nx][t] = false
					wf.updateEntropy(ny, nx)
					changed = true
				}
			}

			if changed {
				stack = append(stack, [2]int{nx, ny})
			}
		}
	}
}

func (wf *WaveFunction) Run() [][]int {
	fmt.Println("Starting generation...")
	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		fmt.Printf("Attempt %d\n", attempt+1)
		wf.reset()

		for !wf.isFullyCollapsed() {
			x, y := wf.findMinEntropyCell()
			if x == -1 {
				fmt.Println("No cell with entropy found, using fallback")
				// Находим первую неколлапсированную клетку
				for y := range wf.collapsed {
					for x := range wf.collapsed[y] {
						if wf.collapsed[y][x] == -1 {
							wf.setTile(x, y, TileGrass)
							break
						}
					}
				}
				continue
			}

			if !wf.collapseCell(x, y) {
				fmt.Printf("Failed to collapse cell (%d,%d)\n", x, y)
				wf.setTile(x, y, TileGrass) // Фолбэк
			}
			wf.propagate(x, y)
		}

		fmt.Println("Generation completed successfully")
		return wf.getResultGrid()
	}

	fmt.Println("Max attempts reached, returning partial result")
	return wf.getResultGrid()
}

func (wf *WaveFunction) reset() {
	for y := range wf.grid {
		for x := range wf.grid[y] {
			for t := range wf.grid[y][x] {
				wf.grid[y][x][t] = true
			}
			wf.collapsed[y][x] = -1
			wf.entropy[y][x] = wf.calculateInitialEntropy(x, y)
		}
	}
}
func (wf *WaveFunction) isFullyCollapsed() bool {
	for y := range wf.collapsed {
		for x := range wf.collapsed[y] {
			if wf.collapsed[y][x] == -1 {
				return false
			}
		}
	}
	return true
}

func (wf *WaveFunction) countNeighbors(x, y, neighborID, maxDist int) int {
	count := 0
	for dy := -maxDist; dy <= maxDist; dy++ {
		for dx := -maxDist; dx <= maxDist; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && ny >= 0 && nx < wf.options.Width && ny < wf.options.Height {
				if wf.collapsed[ny][nx] == neighborID {
					count++
				}
			}
		}
	}
	return count
}

func (wf *WaveFunction) getResultGrid() [][]int {
	grid := make([][]int, len(wf.grid))
	for y := range wf.grid {
		grid[y] = make([]int, len(wf.grid[y]))
		for x := range wf.grid[y] {
			if wf.collapsed[y][x] == -1 {
				grid[y][x] = 0
			} else {
				grid[y][x] = wf.collapsed[y][x]
			}
		}
	}
	return grid
}

// updateEntropy пересчитывает энтропию для указанной клетки
func (wf *WaveFunction) updateEntropy(y, x int) {
	if wf.collapsed[y][x] != -1 {
		wf.entropy[y][x] = 0
		return
	}

	entropy := 0.0
	for t, allowed := range wf.grid[y][x] {
		if allowed {
			weight := wf.urbanWeights[t]

			// Плавное уменьшение веса для больших зданий
			if wf.tiles[t].Size > 1 {
				if !wf.canPlaceLargeTile(x, y, wf.tiles[t].Size) {
					continue
				}
				weight *= math.Pow(1.1, float64(wf.tiles[t].Size))
			}

			// Более мягкое влияние кластеров
			if wf.tiles[t].ClusterSize > 1 {
				sameNeighbors := wf.countNeighbors(x, y, wf.tiles[t].ID, wf.tiles[t].ClusterSize)
				weight *= 1.0 + float64(sameNeighbors)*0.15
			}

			// Более строгое минимальное расстояние
			if wf.tiles[t].MinDistance > 0 {
				sameInRadius := wf.countNeighbors(x, y, wf.tiles[t].ID, wf.tiles[t].MinDistance)
				if sameInRadius > 0 {
					weight *= 0.1
				}
			}

			entropy += weight
		}
	}

	wf.entropy[y][x] = entropy * (0.9 + wf.rng.Float64()*0.2)
}

// getPossibleTiles возвращает список возможных тайлов для клетки
func (wf *WaveFunction) getPossibleTiles(x, y int) []int {
	var possible []int

	for t, allowed := range wf.grid[y][x] {
		if !allowed {
			continue
		}

		tile := wf.tiles[t]

		// Для больших зданий проверяем, поместятся ли они
		if tile.Size > 1 {
			if !wf.canPlaceLargeTile(x, y, tile.Size) {
				continue
			}
		}

		possible = append(possible, t)
	}

	return possible
}

// Вспомогательная функция для максимума
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (wf *WaveFunction) findMinEntropyCell() (int, int) {
	minEntropy := math.MaxFloat64
	var candidates [][2]int

	for y := range wf.entropy {
		for x := range wf.entropy[y] {
			if wf.collapsed[y][x] != -1 {
				continue
			}

			// Пересчитываем энтропию на лету
			currentEntropy := 0.0
			for t := range wf.tiles {
				if wf.grid[y][x][t] {
					currentEntropy += wf.tiles[t].Weight
				}
			}

			if currentEntropy <= 0 {
				continue
			}

			if currentEntropy < minEntropy {
				minEntropy = currentEntropy
				candidates = [][2]int{{x, y}}
			} else if currentEntropy == minEntropy {
				candidates = append(candidates, [2]int{x, y})
			}
		}
	}

	if len(candidates) == 0 {
		// Если не найдено подходящих клеток, выбираем первую незаполненную
		for y := range wf.collapsed {
			for x := range wf.collapsed[y] {
				if wf.collapsed[y][x] == -1 {
					return x, y
				}
			}
		}
		return -1, -1
	}

	return candidates[wf.rng.Intn(len(candidates))][0], candidates[wf.rng.Intn(len(candidates))][1]
}
