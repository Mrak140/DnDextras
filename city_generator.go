package main

import (
	"math/rand"
	"test/internal/wfc"
)

const (
	citySize     = 50
	urbanDensity = 0.45
)

type CityGenerator struct {
	seed int64
	rng  *rand.Rand
}

func NewCityGenerator(seed int64) *CityGenerator {
	return &CityGenerator{
		seed: seed,
		rng:  rand.New(rand.NewSource(seed)),
	}
}

func (cg *CityGenerator) Generate() [][]int {
	tiles := wfc.CreateOrganicCityTiles()

	options := wfc.WFCOptions{
		Width:        citySize,
		Height:       citySize,
		Tiles:        tiles,
		Seed:         cg.seed,
		Iterations:   30000,
		UrbanDensity: urbanDensity,
	}

	wfc := wfc.NewWFC(options)
	grid := wfc.Run()

	cg.organicPostProcessing(grid)
	return grid
}

func (cg *CityGenerator) organicPostProcessing(grid [][]int) {
	cg.smoothWaterFeatures(grid)
	cg.formBuildingClusters(grid)
	cg.addNaturalDetails(grid)
}

func (cg *CityGenerator) smoothWaterFeatures(grid [][]int) {
	// Делаем границы воды более естественными
	for y := 1; y < len(grid)-1; y++ {
		for x := 1; x < len(grid[0])-1; x++ {
			if grid[y][x] == wfc.TileWater {
				waterNeighbors := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if grid[y+dy][x+dx] == wfc.TileWater {
							waterNeighbors++
						}
					}
				}
				// Если вода слишком изолирована, убираем её
				if waterNeighbors < 3 && cg.rng.Float32() < 0.7 {
					grid[y][x] = wfc.TileGrass
				}
			} else if grid[y][x] == wfc.TileGrass {
				// Добавляем небольшие водоёмы
				if cg.countNeighbors(grid, x, y, wfc.TileWater) >= 5 &&
					cg.rng.Float32() < 0.4 {
					grid[y][x] = wfc.TileWater
				}
			}
		}
	}
}

func (cg *CityGenerator) formBuildingClusters(grid [][]int) {
	// Усиливаем кластеры зданий
	for y := 1; y < len(grid)-1; y++ {
		for x := 1; x < len(grid[0])-1; x++ {
			tile := grid[y][x]
			if tile == wfc.TileResidential1x1 || tile == wfc.TileResidential2x2 {
				buildingNeighbors := cg.countNeighbors(grid, x, y, tile)

				// Если здание в хорошем кластере, иногда добавляем рядом ещё
				if buildingNeighbors >= 2 && cg.rng.Float32() < 0.3 {
					cg.addAdjacentBuilding(grid, x, y, tile)
				}

				// Если здание слишком одинокое, убираем его
				if buildingNeighbors == 0 && cg.rng.Float32() < 0.8 {
					grid[y][x] = wfc.TileGrass
				}
			}
		}
	}
}

func (cg *CityGenerator) addNaturalDetails(grid [][]int) {
	// Добавляем мелкие детали для естественности
	for y := 0; y < len(grid); y++ {
		for x := 0; x < len(grid[0]); x++ {
			if grid[y][x] == wfc.TileGrass && cg.rng.Float32() < 0.05 {
				// Небольшие декоративные элементы
				grid[y][x] = wfc.TileEmpty
			}
		}
	}
}

// Вспомогательные методы
func (cg *CityGenerator) countNeighbors(grid [][]int, x, y, tileType int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && ny >= 0 && nx < len(grid[0]) && ny < len(grid) {
				if grid[ny][nx] == tileType {
					count++
				}
			}
		}
	}
	return count
}

func (cg *CityGenerator) addAdjacentBuilding(grid [][]int, x, y, buildingType int) {
	// Пытаемся добавить здание рядом
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}
	cg.rng.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})

	for _, dir := range directions {
		nx, ny := x+dir[0], y+dir[1]
		if nx >= 0 && ny >= 0 && nx < len(grid[0]) && ny < len(grid) {
			if grid[ny][nx] == wfc.TileGrass {
				grid[ny][nx] = buildingType
				break
			}
		}
	}
}
