package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

const (
	characterWindowWidth  = 500
	characterWindowHeight = 600
	buttonHeight          = 30
	buttonWidth           = 80
	navButtonWidth        = 40
)

// Вспомогательные функции
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Выносим отрисовку статистики в отдельный метод
func (g *Game) drawCharacterStats(screen *ebiten.Image, char *Character, padding int) {
	// Основная информация
	infoY := 60
	text.Draw(screen, fmt.Sprintf("Раса: %s", char.Race), g.font, padding, infoY, color.White)
	text.Draw(screen, fmt.Sprintf("Предыстория: %s", char.Background), g.font, padding, infoY+20, color.White)

	// Характеристики
	statsY := infoY + 60
	text.Draw(screen, "Характеристики:", g.font, padding, statsY, color.RGBA{255, 255, 0, 255})
	statsY += 20

	// Оптимизация: используем итерацию по карте для характеристик
	statsOrder := []string{"str", "dex", "con", "int", "wis", "cha"}
	statNames := map[string]string{
		"str": "Сила", "dex": "Ловкость", "con": "Телосложение",
		"int": "Интеллект", "wis": "Мудрость", "cha": "Харизма",
	}

	for i, stat := range statsOrder {
		x := padding + (i%3)*140
		y := statsY + (i/3)*20
		text.Draw(screen, fmt.Sprintf("%s: %d", statNames[stat], char.Stats[stat]),
			g.font, x, y, color.White)
	}
}

// wrapText разбивает текст на строки по ширине
func wrapText(textStr string, maxWidth int, font font.Face) []string {
	var result []string
	if font == nil {
		return result
	}

	words := strings.Fields(textStr)
	if len(words) == 0 {
		return result
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		testLine := currentLine + " " + word
		bounds := text.BoundString(font, testLine)
		if bounds.Max.X <= maxWidth {
			currentLine = testLine
		} else {
			// Если даже одно слово не помещается, разбиваем его
			if currentLine == "" {
				result = append(result, word)
				continue
			}
			result = append(result, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		result = append(result, currentLine)
	}
	return result
}

func (g *Game) handleCharacterWindowInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// Проверка кнопки закрытия
		buttonX := characterWindowWidth - buttonWidth - 10
		buttonY := characterWindowHeight - buttonHeight - 10
		if mx >= buttonX && mx <= buttonX+buttonWidth &&
			my >= buttonY && my <= buttonY+buttonHeight {
			g.characterWindowOpen = false
			return
		}

		// Проверка кнопок навигации (если персонажей больше одного)
		if len(g.characters) > 1 {
			prevBtnX := 10
			nextBtnX := characterWindowWidth - navButtonWidth - 10
			navBtnY := characterWindowHeight - buttonHeight - 10

			if mx >= prevBtnX && mx <= prevBtnX+navButtonWidth &&
				my >= navBtnY && my <= navBtnY+buttonHeight {
				g.characterIndex--
				if g.characterIndex < 0 {
					g.characterIndex = len(g.characters) - 1
				}
				g.currentCharacter = g.characters[g.characterIndex]
			}

			if mx >= nextBtnX && mx <= nextBtnX+navButtonWidth &&
				my >= navBtnY && my <= navBtnY+buttonHeight {
				g.characterIndex++
				if g.characterIndex >= len(g.characters) {
					g.characterIndex = 0
				}
				g.currentCharacter = g.characters[g.characterIndex]
			}
		}
	}
}

func (g *Game) drawCharacterDescription(screen *ebiten.Image, char *Character, padding int) {
	descY := 200 // Начальная позиция по Y
	text.Draw(screen, "Описание:", g.font, padding, descY, color.RGBA{255, 255, 0, 255})
	descY += 20

	// Разбиваем описание на строки для правильного отображения
	descLines := strings.Split(char.Description, "\n")
	for _, line := range descLines {
		if line == "" {
			continue
		}
		wrapped := wrapText(line, characterWindowWidth-padding*2, g.font)
		for _, wrappedLine := range wrapped {
			if descY > characterWindowHeight-60 { // Оставляем место для кнопки
				break
			}
			text.Draw(screen, wrappedLine, g.font, padding, descY, color.White)
			descY += 20
		}
	}
}

// drawNavigationButtons рисует кнопки навигации между персонажами
func (g *Game) drawNavigationButtons(screen *ebiten.Image) {
	if len(g.characters) <= 1 {
		return
	}

	// Кнопка "Назад"
	prevBtnX := 10
	prevBtnY := characterWindowHeight - buttonHeight - 10
	ebitenutil.DrawRect(screen, float64(prevBtnX), float64(prevBtnY), navButtonWidth, buttonHeight, color.RGBA{70, 70, 90, 255})
	text.Draw(screen, "<", g.font, prevBtnX+15, prevBtnY+20, color.White)

	// Информация о текущем персонаже
	charInfo := fmt.Sprintf("%d/%d", g.characterIndex+1, len(g.characters))
	text.Draw(screen, charInfo, g.font, characterWindowWidth/2-10, prevBtnY+20, color.White)

	// Кнопка "Вперед"
	nextBtnX := characterWindowWidth - navButtonWidth - 10
	nextBtnY := prevBtnY
	ebitenutil.DrawRect(screen, float64(nextBtnX), float64(nextBtnY), navButtonWidth, buttonHeight, color.RGBA{70, 70, 90, 255})
	text.Draw(screen, ">", g.font, nextBtnX+15, nextBtnY+20, color.White)

	// Кнопка закрытия
	closeBtnX := characterWindowWidth - buttonWidth - 10
	closeBtnY := prevBtnY
	ebitenutil.DrawRect(screen, float64(closeBtnX), float64(closeBtnY), buttonWidth, buttonHeight, color.RGBA{100, 0, 0, 255})
	text.Draw(screen, "Закрыть", g.font, closeBtnX+10, closeBtnY+20, color.White)
}

// Обновляем метод drawCharacterWindow для использования новых методов
func (g *Game) drawCharacterWindow(screen *ebiten.Image) {
	if g.currentCharacter == nil || g.font == nil {
		return
	}

	char := g.currentCharacter
	padding := 20

	// Фон окна
	ebitenutil.DrawRect(screen, 0, 0, characterWindowWidth, characterWindowHeight, color.RGBA{30, 30, 40, 230})

	// Заголовок (Имя - Класс Уровня)
	title := fmt.Sprintf("%s - %s %d уровня", char.Name, char.Class, char.Level)
	bounds := text.BoundString(g.font, title)
	titleX := max(characterWindowWidth/2-bounds.Max.X/2, padding)
	text.Draw(screen, title, g.font, titleX, 30, color.White)

	// Отрисовка характеристик
	g.drawCharacterStats(screen, char, padding)

	// Отрисовка описания
	g.drawCharacterDescription(screen, char, padding)

	// Отрисовка кнопок навигации
	g.drawNavigationButtons(screen)
}
