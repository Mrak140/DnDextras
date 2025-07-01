package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func (g *Game) initCityMap(city *City) {
	g.mu.Lock()
	defer g.mu.Unlock()

	generator := NewCityGenerator(int64(city.Population))
	cityGrid := generator.Generate()

	g.cityMap = &CityMap{
		City:      city,
		Tiles:     make([][]color.Color, cityMapSize),
		Buildings: make([]Building, 0),
		Open:      true,
		Grid:      cityGrid,
	}

	for y := 0; y < cityMapSize; y++ {
		g.cityMap.Tiles[y] = make([]color.Color, cityMapSize)
		for x := 0; x < cityMapSize; x++ {
			if y < len(cityGrid) && x < len(cityGrid[y]) {
				g.cityMap.Tiles[y][x] = g.getEnhancedTileColor(cityGrid, x, y)
			} else {
				g.cityMap.Tiles[y][x] = color.RGBA{80, 80, 80, 255}
			}
		}
	}

	g.generateStructuredBuildings(cityGrid)
}

func (g *Game) generateStructuredBuildings(grid [][]int) {
	g.cityMap.Buildings = make([]Building, 0)

	for y := 0; y < len(grid) && y < cityMapSize; y++ {
		for x := 0; x < len(grid[y]) && x < cityMapSize; x++ {
			tile := grid[y][x]

			if tile == TileResidential || tile == TileCommercial {
				building := Building{
					X:     x,
					Y:     y,
					Type:  getBuildingType(tile),
					Level: g.calculateBuildingLevel(grid, x, y),
				}
				g.cityMap.Buildings = append(g.cityMap.Buildings, building)
			}
		}
	}
}

func getBuildingType(tile int) string {
	switch tile {
	case TileResidential:
		return "residential"
	case TileCommercial:
		return "commercial"
	default:
		return ""
	}
}

func (g *Game) calculateBuildingLevel(grid [][]int, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < len(grid[0]) && ny >= 0 && ny < len(grid) {
				if grid[ny][nx] == grid[y][x] {
					count++
				}
			}
		}
	}

	switch {
	case count > 6:
		return 3
	case count > 3:
		return 2
	default:
		return 1
	}
}

func (g *Game) getEnhancedTileColor(grid [][]int, x, y int) color.RGBA {
	baseColor := getBaseTileColor(grid[y][x])
	variation := (x + y) % 5

	switch grid[y][x] {
	case TileResidential:
		return color.RGBA{
			R: uint8(int(baseColor.R) + variation*3),
			G: uint8(int(baseColor.G) + variation*2),
			B: baseColor.B,
			A: 255,
		}
	case TileCommercial:
		return color.RGBA{
			R: uint8(int(baseColor.R) + variation*4),
			G: uint8(int(baseColor.G) + variation),
			B: uint8(int(baseColor.B) + variation*2),
			A: 255,
		}
	default:
		return baseColor
	}
}

func getBaseTileColor(tile int) color.RGBA {
	switch tile {
	case TileRoad:
		return color.RGBA{255, 255, 255, 255} // Белые дороги
	case TileResidential:
		return color.RGBA{180, 180, 180, 255}
	case TileCommercial:
		return color.RGBA{120, 90, 90, 255}
	case TilePark:
		return color.RGBA{50, 200, 50, 255}
	case TileWater:
		return color.RGBA{70, 140, 210, 255}
	case TileSpecial:
		return color.RGBA{150, 150, 0, 255}
	case TilePlaza:
		return color.RGBA{200, 180, 120, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}

func (g *Game) updateCityMap() {
	if g.cityMap == nil || !g.cityMap.Open {
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		buttonX := cityMapSize*buildingSize*cityMapScale - 80
		buttonY := cityMapSize*buildingSize*cityMapScale + 10
		buttonWidth := 70
		buttonHeight := 20

		if mx >= buttonX && mx <= buttonX+buttonWidth &&
			my >= buttonY && my <= buttonY+buttonHeight {
			g.cityMap.Open = false
		}
	}
}

func (g *Game) drawCityMap(screen *ebiten.Image) {
	if g.cityMap == nil || !g.cityMap.Open {
		return
	}

	// Draw tiles
	for y := 0; y < cityMapSize; y++ {
		for x := 0; x < cityMapSize; x++ {
			col := getBaseTileColor(g.cityMap.Grid[y][x])
			ebitenutil.DrawRect(screen,
				float64(x*buildingSize*cityMapScale),
				float64(y*buildingSize*cityMapScale),
				buildingSize*cityMapScale,
				buildingSize*cityMapScale,
				col)
		}
	}

	// Draw city name
	text.Draw(screen, g.cityMap.City.Name, g.font, 10, 20, color.White)
}

func getBuildingColor(buildingType string) color.RGBA {
	switch buildingType {
	case "commercial":
		return color.RGBA{120, 90, 90, 255}
	case "special":
		return color.RGBA{150, 150, 0, 255}
	default: // residential
		return color.RGBA{180, 180, 180, 255} // Светло-серый для жилых
	}
}
