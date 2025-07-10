package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type CharacterFile struct {
	Data   string `json:"data"` // Теперь обрабатываем как строку
	Spells struct {
		Prepared []string `json:"prepared"`
		Book     []string `json:"book"`
	} `json:"spells"`
}

type CharacterData struct {
	Name struct {
		Value string `json:"value"`
	} `json:"name"`
	Info struct {
		CharClass struct {
			Value string `json:"value"`
		} `json:"charClass"`
		Level struct {
			Value interface{} `json:"value"` // Может быть int или string
		} `json:"level"`
		Race struct {
			Value string `json:"value"`
		} `json:"race"`
		Background struct {
			Value string `json:"value"`
		} `json:"background"`
	} `json:"info"`
	Stats map[string]struct {
		Score int `json:"score"`
	} `json:"stats"`
	Vitality struct {
		HPCurrent struct {
			Value interface{} `json:"value"` // Может быть int или string
		} `json:"hp-current"`
		AC struct {
			Value interface{} `json:"value"` // Может быть int или string
		} `json:"ac"`
	} `json:"vitality"`
	Text struct {
		Background struct {
			Value json.RawMessage `json:"value"`
		} `json:"background"`
		Notes1 struct {
			Value json.RawMessage `json:"value"`
		} `json:"notes-1"`
	} `json:"text"`
}

func (g *Game) loadCharacterFromFile(filename string) (*Character, error) {
	if debugMode {
		log.Printf("[DEBUG] Начало загрузки персонажа из файла: %s", filename)
	}

	fullPath := filepath.Clean(filename)
	if debugMode {
		log.Printf("[DEBUG] Обработка файла по пути: %s", fullPath)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("[ERROR] Ошибка чтения файла %s: %v", fullPath, err)
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Удаляем BOM маркер если есть
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	if debugMode && len(data) > 0 {
		log.Printf("[DEBUG] Прочитано %d байт из файла", len(data))
	}

	var raw CharacterFile
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[ERROR] Ошибка парсинга JSON из файла %s: %v", fullPath, err)
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if debugMode {
		log.Printf("[DEBUG] Успешно распарсен файл, начало обработки данных персонажа")
	}

	// Теперь парсим вложенный JSON из поля data
	var charData CharacterData
	if err := json.Unmarshal([]byte(raw.Data), &charData); err != nil {
		log.Printf("[ERROR] Ошибка парсинга вложенного JSON: %v", err)
		return nil, fmt.Errorf("ошибка парсинга вложенного JSON: %v", err)
	}

	char := &Character{
		Name:       charData.Name.Value,
		Class:      charData.Info.CharClass.Value,
		Race:       charData.Info.Race.Value,
		Background: charData.Info.Background.Value,
		Stats:      make(map[string]int),
		Spells:     append(raw.Spells.Prepared, raw.Spells.Book...),
		Skills:     []string{},
		Equipment:  []string{},
	}

	if debugMode {
		log.Printf("[DEBUG] Создан базовый объект персонажа: %s", char.Name)
	}

	// Обрабатываем уровень
	switch v := charData.Info.Level.Value.(type) {
	case float64:
		char.Level = int(v)
	case string:
		fmt.Sscanf(v, "%d", &char.Level)
	}
	if debugMode {
		log.Printf("[DEBUG] Установлен уровень персонажа: %d", char.Level)
	}

	// Обрабатываем HP
	switch v := charData.Vitality.HPCurrent.Value.(type) {
	case float64:
		char.HP = fmt.Sprintf("%d", int(v))
	case string:
		char.HP = v
	}

	// Обрабатываем AC
	switch v := charData.Vitality.AC.Value.(type) {
	case float64:
		char.AC = fmt.Sprintf("%d", int(v))
	case string:
		char.AC = v
	}

	if debugMode {
		log.Printf("[DEBUG] Установлены HP: %s и AC: %s", char.HP, char.AC)
	}

	// Заполняем характеристики
	for stat, data := range charData.Stats {
		char.Stats[stat] = data.Score
	}
	if debugMode {
		log.Printf("[DEBUG] Установлены характеристики: %+v", char.Stats)
	}

	// Обрабатываем описание
	char.Description = extractTextContent(charData.Text.Background.Value)
	if debugMode {
		log.Printf("[DEBUG] Длина описания: %d символов", len(char.Description))
	}

	// Обрабатываем заметки
	char.Notes = extractTextContent(charData.Text.Notes1.Value)
	if debugMode {
		log.Printf("[DEBUG] Длина заметок: %d символов", len(char.Notes))
		log.Printf("[DEBUG] Успешно загружен персонаж: %s", char.Name)
	}

	return char, nil
}

func extractTextContent(raw json.RawMessage) string {
	// Пытаемся распарсить как сложную структуру
	var complexStruct struct {
		Data struct {
			Content []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"content"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &complexStruct); err == nil {
		var builder strings.Builder
		for _, content := range complexStruct.Data.Content {
			for _, textContent := range content.Content {
				if textContent.Text != "" {
					builder.WriteString(textContent.Text)
					builder.WriteString("\n")
				}
			}
		}
		return strings.TrimSpace(builder.String())
	}

	// Пытаемся распарсить как простую строку
	var simpleStr string
	if err := json.Unmarshal(raw, &simpleStr); err == nil {
		return simpleStr
	}

	return ""
}

func (g *Game) loadAllCharacters() error {
	const charactersDir = "characters"
	log.Printf("Начало загрузки персонажей из: %s", charactersDir)

	if _, err := os.Stat(charactersDir); os.IsNotExist(err) {
		msg := fmt.Sprintf("Директория не найдена: %s", charactersDir)
		log.Printf("[ERROR] %s", msg)
		return fmt.Errorf(msg)
	}

	files, err := os.ReadDir(charactersDir)
	if err != nil {
		log.Printf("[ERROR] Ошибка чтения директории %s: %v", charactersDir, err)
		return fmt.Errorf("ошибка чтения директории: %v", err)
	}

	if debugMode {
		log.Printf("[DEBUG] Найдено %d файлов в директории", len(files))
	}

	var loadedChars int
	for _, file := range files {
		if file.IsDir() {
			if debugMode {
				log.Printf("[DEBUG] Пропускаем директорию: %s", file.Name())
			}
			continue
		}

		if !strings.HasSuffix(file.Name(), ".json") {
			if debugMode {
				log.Printf("[DEBUG] Пропускаем файл с неподходящим расширением: %s", file.Name())
			}
			continue
		}

		fullPath := filepath.Join(charactersDir, file.Name())
		if debugMode {
			log.Printf("[DEBUG] Обработка файла персонажа: %s", fullPath)
		}

		char, err := g.loadCharacterFromFile(fullPath)
		if err != nil {
			log.Printf("[ERROR] Ошибка загрузки %s: %v", file.Name(), err)
			continue
		}

		g.characters = append(g.characters, char)
		loadedChars++
		log.Printf("Успешно загружен персонаж %d: %s", loadedChars, char.Name)
	}

	if loadedChars == 0 {
		msg := "Не найдено валидных файлов персонажей"
		log.Printf("[ERROR] %s", msg)
		return fmt.Errorf(msg)
	}

	g.currentCharacter = g.characters[0]
	g.characterIndex = 0
	log.Printf("Всего загружено персонажей: %d", loadedChars)

	if debugMode {
		log.Printf("[DEBUG] Текущий персонаж: %s (индекс %d)",
			g.currentCharacter.Name, g.characterIndex)
		log.Printf("[DEBUG] Список загруженных персонажей:")
		for i, ch := range g.characters {
			log.Printf("[DEBUG] %d: %s - %s %d уровня",
				i+1, ch.Name, ch.Class, ch.Level)
		}
	}

	return nil
}
