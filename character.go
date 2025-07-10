package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func (g *Game) loadCharacterFromJSON(filename string) (*Character, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	char := &Character{
		Stats:     make(map[string]int),
		Skills:    []string{},
		Equipment: []string{},
		Spells:    []string{},
	}

	// Безопасное извлечение данных
	if data, ok := rawData["data"].(map[string]interface{}); ok {
		// Имя персонажа
		if name, ok := data["name"].(map[string]interface{}); ok {
			if val, ok := name["value"].(string); ok {
				char.Name = val
			}
		}

		// Основная информация
		if info, ok := data["info"].(map[string]interface{}); ok {
			if class, ok := info["charClass"].(map[string]interface{}); ok {
				if val, ok := class["value"].(string); ok {
					char.Class = val
				}
			}
			if level, ok := info["level"].(map[string]interface{}); ok {
				if val, ok := level["value"].(float64); ok {
					char.Level = int(val)
				}
			}
			if race, ok := info["race"].(map[string]interface{}); ok {
				if val, ok := race["value"].(string); ok {
					char.Race = val
				}
			}
			if bg, ok := info["background"].(map[string]interface{}); ok {
				if val, ok := bg["value"].(string); ok {
					char.Background = val
				}
			}
		}

		// Характеристики
		if stats, ok := data["stats"].(map[string]interface{}); ok {
			for stat, value := range stats {
				if statMap, ok := value.(map[string]interface{}); ok {
					if score, ok := statMap["score"].(float64); ok {
						char.Stats[stat] = int(score)
					}
				}
			}
		}

		// Описание
		if textData, ok := data["text"].(map[string]interface{}); ok {
			if bg, ok := textData["background"].(map[string]interface{}); ok {
				if val, ok := bg["value"]; ok {
					switch v := val.(type) {
					case string:
						char.Description = v
					case map[string]interface{}:
						if data, ok := v["data"].(map[string]interface{}); ok {
							if content, ok := data["content"].([]interface{}); ok {
								for _, item := range content {
									if itemMap, ok := item.(map[string]interface{}); ok {
										if content, ok := itemMap["content"].([]interface{}); ok {
											for _, textItem := range content {
												if textMap, ok := textItem.(map[string]interface{}); ok {
													if text, ok := textMap["text"].(string); ok {
														char.Description += text + "\n"
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return char, nil
}

func (g *Game) toggleCharacterWindow() {
	if g == nil {
		log.Printf("[ERROR] Game объект nil в toggleCharacterWindow")
		return
	}

	if debugMode {
		log.Printf("[DEBUG] toggleCharacterWindow: текущее состояние %v", g.characterWindowOpen)
	}

	g.characterWindowOpen = !g.characterWindowOpen

	if g.characterWindowOpen {
		if len(g.characters) == 0 {
			log.Printf("[WARN] Нет загруженных персонажей, попытка загрузки...")
			if err := g.loadAllCharacters(); err != nil {
				log.Printf("[ERROR] Ошибка загрузки персонажей: %v", err)
				g.characterWindowOpen = false
				return
			}
		}

		if debugMode {
			log.Printf("[DEBUG] Открытие окна персонажа для: %s", g.currentCharacter.Name)
		}
	} else if debugMode {
		log.Printf("[DEBUG] Закрытие окна персонажа")
	}
}
