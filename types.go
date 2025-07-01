package main

import "image/color"

const (
	screenWidth                 = 1920
	screenHeight                = 1080
	cellSize                    = 16
	serverPort                  = ":8080"
	cityMapSize                 = 35
	buildingSize                = 8
	cityMapScale                = 4
	TileStone                         // Новый тип тайла для каменных блоков
	buildingSizeMultiplier      = 1.5 // Увеличиваем размер зданий
	TileEmpty                   = 0
	TileRoad                    = 1
	TileResidential             = 2
	TileCommercial              = 3
	TilePark                    = 4
	TileWater                   = 5
	TilePath                    = 6
	TileSpecial                 = 7
	TilePlaza                   = 8
	TileGrass                   = iota
	minDistanceBetweenBuildings = 3
	maxBuildingLevel            = 5
	BuildingSmall               = 1    // 1x1
	BuildingMedium              = 2    // 2x2
	BuildingLarge               = 3    // 3x3 и более
	specialBuildingChance       = 0.05 // 5% шанс генерации специального здания
	debugMode                   = true
)

type Player struct {
	ID    string
	X, Y  int
	Color color.RGBA
}

type Character struct {
	Name        string
	Class       string
	Level       int
	Race        string
	Background  string
	Stats       map[string]int
	HP          string
	AC          string
	Description string
	Notes       string
	Spells      []string
	Skills      []string
	Equipment   []string
}

type City struct {
	Name       string
	X, Y       int
	Size       int
	Population int
}

type CityWindow struct {
	game       *Game
	open       bool
	selected   *City
	scrollY    int
	hoverIndex int
	cities     []*City
}

type CityMap struct {
	City      *City
	Tiles     [][]color.Color
	Buildings []Building
	Open      bool
	Grid      [][]int
}

type Building struct {
	X, Y   int    // Левый верхний угол
	Width  int    // Ширина в тайлах
	Height int    // Высота в тайлах
	Type   string // residential/commercial
	Level  int    // Этажность (1-5)
}

type point struct {
	x, y int
}

type RoadNetwork struct {
	grid      [][]int
	buildings []Building
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
