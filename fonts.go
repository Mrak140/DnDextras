package main

import (
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

func loadTrueTypeFont(path string, size float64) font.Face {
	fontData, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Ошибка загрузки шрифта: %v", err)
		return basicfont.Face7x13
	}

	tt, err := opentype.Parse(fontData)
	if err != nil {
		log.Printf("Ошибка парсинга шрифта: %v", err)
		return basicfont.Face7x13
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingVertical, // Используем доступный вариант
	})
	if err != nil {
		log.Printf("Ошибка создания шрифта: %v", err)
		return basicfont.Face7x13
	}

	return face
}
