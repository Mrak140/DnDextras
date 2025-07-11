// temperature_map.go
package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/aquilax/go-perlin"
)

type Temperature int

const (
	Frozen Temperature = iota
	Cold
	Cool
	Warm
	Hot
)

type TemperatureMap struct {
	Grid [][]Temperature
	seed int64
}

func NewTemperatureMap(seed int64, width, height int) *TemperatureMap {
	tm := &TemperatureMap{
		seed: seed,
	}
	if seed == 0 {
		tm.seed = time.Now().UnixNano()
	}
	tm.Generate(width, height)
	return tm
}

func (tm *TemperatureMap) Generate(width, height int) {
	rand.Seed(tm.seed)
	p := perlin.NewPerlin(2, 2, 5, tm.seed)

	tm.Grid = make([][]Temperature, height)

	// Генерируем шум и находим min/max для нормализации
	min, max := math.MaxFloat64, -math.MaxFloat64
	noiseValues := make([][]float64, height)
	for y := 0; y < height; y++ {
		noiseValues[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			nx := float64(x) / float64(width) * 6
			ny := float64(y) / float64(height) * 6
			val := p.Noise2D(nx, ny)
			noiseValues[y][x] = val
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	}

	// Нормализуем значения и создаем температурную сетку
	for y := 0; y < height; y++ {
		tm.Grid[y] = make([]Temperature, width)
		for x := 0; x < width; x++ {
			normalized := (noiseValues[y][x] - min) / (max - min)

			// Распределяем температуры с учетом климатических зон
			switch {
			case normalized < 0.1: // 10% - Frozen
				tm.Grid[y][x] = Frozen
			case normalized < 0.25: // 15% - Cold
				tm.Grid[y][x] = Cold
			case normalized < 0.45: // 20% - Cool
				tm.Grid[y][x] = Cool
			case normalized < 0.75: // 30% - Warm
				tm.Grid[y][x] = Warm
			default: // 20% - Hot
				tm.Grid[y][x] = Hot
			}
		}
	}
}

func (tm *TemperatureMap) GetTemperature(x, y int) Temperature {
	if x < 0 || y < 0 || y >= len(tm.Grid) || x >= len(tm.Grid[0]) {
		return Cool // Возвращаем значение по умолчанию
	}
	return tm.Grid[y][x]
}
