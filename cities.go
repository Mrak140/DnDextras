package main

import (
	"fmt"
	"math/rand"
)

var cityNames = []string{
	"Новоград", "Речной", "Солнечный", "Лесной", "Морской",
	"Старый", "Северный", "Южный", "Восточный", "Западный",
	"Центральный", "Портовый", "Горный", "Долинный", "Озёрный",
}

func (g *Game) generateCities() {
	if g.noiseMap == nil || len(g.noiseMap) == 0 {
		g.generateWorld() // Если карта не сгенерирована, создаём её
	}

	cols := screenWidth / cellSize
	rows := screenHeight / cellSize

	g.cities = make([][]bool, rows)
	for y := 0; y < rows; y++ {
		g.cities[y] = make([]bool, cols)
	}

	g.cityList = make([]*City, 0)

	// Увеличим шанс генерации городов
	for i := 0; i < 50; i++ { // Попробуем сгенерировать 50 городов
		x := rand.Intn(cols-10) + 5
		y := rand.Intn(rows-10) + 5

		biomeValue := (g.noiseMap[y][x] + 1) / 2
		if biomeValue > 0.5 && biomeValue < 0.8 { // Только на подходящих биомах
			g.createCityAt(x, y)
		}
	}

	fmt.Printf("Сгенерировано %d городов\n", len(g.cityList)) // Отладочный вывод
}
func (g *Game) createCityAt(x, y int) {
	city := &City{
		Name:       generateCityName(),
		X:          x,
		Y:          y,
		Size:       rand.Intn(3) + 1,
		Population: rand.Intn(90000) + 10000,
	}
	g.cityList = append(g.cityList, city)

	cols := screenWidth / cellSize
	rows := screenHeight / cellSize
	for dy := -city.Size; dy <= city.Size; dy++ {
		for dx := -city.Size; dx <= city.Size; dx++ {
			ny, nx := y+dy, x+dx
			if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
				g.cities[ny][nx] = true
			}
		}
	}
}

func generateCityName() string {
	name1 := cityNames[rand.Intn(len(cityNames))]
	name2 := cityNames[rand.Intn(len(cityNames))]
	if rand.Float32() > 0.3 {
		return name1 + "-" + name2
	}
	return name1
}

func (g *Game) findCityAt(x, y int) *City {
	for _, city := range g.cityList {
		if x >= city.X-city.Size && x <= city.X+city.Size &&
			y >= city.Y-city.Size && y <= city.Y+city.Size {
			return city
		}
	}
	return nil
}
