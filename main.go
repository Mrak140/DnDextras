package main

import (
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

type Game struct {
	perlin              *Perlin
	noiseMap            [][]float64
	tiles               [][]color.Color
	cities              [][]bool
	mode                string
	conn                net.Conn
	encoder             *gob.Encoder
	decoder             *gob.Decoder
	players             map[string]Player
	mu                  sync.Mutex
	seed                int64
	me                  Player
	cameraX             int
	cameraY             int
	font                font.Face
	cityList            []*City
	hoverCity           *City
	cityWindow          *CityWindow
	cityMap             *CityMap
	cityTemplates       map[int64]*CityMap
	characters          []*Character
	currentCharacter    *Character
	characterIndex      int
	characterWindowOpen bool
}

type Perlin struct {
	permutation []int
	gradients   [][]float64
}

func main() {
	rand.Seed(time.Now().UnixNano())

	game := &Game{
		perlin:   NewPerlin(time.Now().UnixNano()),
		players:  make(map[string]Player),
		font:     loadTrueTypeFont("assets/NotoSans-Regular.ttf", 14), // Загружаем наш шрифт вместо basicfont
		cityList: make([]*City, 0),
		me: Player{
			ID:    fmt.Sprintf("игрок-%d", rand.Intn(1000)),
			Color: randomColor(),
		},
		cameraX: -screenWidth / 4,  // Начальная позиция камеры
		cameraY: -screenHeight / 4, // чтобы видеть больше карты
	}
	game.initCityWindow()

	fmt.Println("Запустить как: [s]erver или [c]lient?")
	var mode string
	_, err := fmt.Scanln(&mode)
	if err != nil {
		log.Fatal("Ошибка ввода:", err)
	}

	switch mode {
	case "s":
		game.mode = "server"
		game.seed = time.Now().UnixNano()
		game.perlin = NewPerlin(game.seed)
		game.generateWorld()
		game.generateCities()
		go game.startServer()

	case "c":
		game.mode = "client"
		game.connectToServer()
	default:
		log.Fatal("Неверный режим. Используйте 's' или 'c'")
	}

	ebiten.SetWindowSize(screenWidth/2, screenHeight/2)
	ebiten.SetWindowTitle(fmt.Sprintf("Сетевой генератор мира (%s) - %s", mode, game.me.ID))
	ebiten.SetWindowResizable(true)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func randomColor() color.RGBA {
	return color.RGBA{
		uint8(rand.Intn(200) + 55),
		uint8(rand.Intn(200) + 55),
		uint8(rand.Intn(200) + 55),
		255,
	}
}

func NewPerlin(seed int64) *Perlin {
	p := &Perlin{
		permutation: make([]int, 256),
		gradients:   make([][]float64, 256),
	}

	r := rand.New(rand.NewSource(seed))

	for i := 0; i < 256; i++ {
		p.permutation[i] = i
	}

	for i := 255; i > 0; i-- {
		j := r.Intn(i + 1)
		p.permutation[i], p.permutation[j] = p.permutation[j], p.permutation[i]
	}

	for i := 0; i < 256; i++ {
		angle := r.Float64() * 2 * math.Pi
		p.gradients[i] = []float64{math.Cos(angle), math.Sin(angle)}
	}

	return p
}

func (p *Perlin) Noise(x, y float64) float64 {
	x0 := int(math.Floor(x)) & 255
	y0 := int(math.Floor(y)) & 255
	x1 := (x0 + 1) & 255
	y1 := (y0 + 1) & 255

	sx := x - math.Floor(x)
	sy := y - math.Floor(y)

	n0 := p.dotGrid(x0, y0, x, y)
	n1 := p.dotGrid(x1, y0, x, y)
	ix0 := p.interpolate(n0, n1, sx)

	n0 = p.dotGrid(x0, y1, x, y)
	n1 = p.dotGrid(x1, y1, x, y)
	ix1 := p.interpolate(n0, n1, sx)

	return p.interpolate(ix0, ix1, sy)
}

func (p *Perlin) dotGrid(ix, iy int, x, y float64) float64 {
	grad := p.gradients[p.permutation[(p.permutation[ix]+iy)%256]]
	dx := x - float64(ix)
	dy := y - float64(iy)
	return dx*grad[0] + dy*grad[1]
}

func (p *Perlin) interpolate(a, b, t float64) float64 {
	t = t * t * t * (t*(t*6-15) + 10)
	return a + t*(b-a)
}

func (g *Game) generateWorld() {
	cols := screenWidth / cellSize
	rows := screenHeight / cellSize
	g.noiseMap = make([][]float64, rows)
	g.tiles = make([][]color.Color, rows)
	g.cities = make([][]bool, rows)

	for y := 0; y < rows; y++ {
		g.noiseMap[y] = make([]float64, cols)
		g.tiles[y] = make([]color.Color, cols)
		g.cities[y] = make([]bool, cols)

		for x := 0; x < cols; x++ {
			nx := float64(x) * 0.05
			ny := float64(y) * 0.05
			g.noiseMap[y][x] = g.perlin.Noise(nx, ny)
			value := (g.noiseMap[y][x] + 1) / 2

			switch {
			case value < 0.4:
				g.tiles[y][x] = color.RGBA{0, 105, 148, 255}
			case value < 0.5:
				g.tiles[y][x] = color.RGBA{194, 178, 128, 255}
			case value < 0.75:
				g.tiles[y][x] = color.RGBA{34, 139, 34, 255}
			case value < 0.95:
				g.tiles[y][x] = color.RGBA{100, 100, 100, 255}
			default:
				g.tiles[y][x] = color.RGBA{220, 220, 220, 255}
			}
		}
	}
}

func (g *Game) startServer() {
	ln, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Println("Сервер запущен на", ln.Addr())

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Ошибка подключения:", err)
			continue
		}

		go g.handleConnection(conn)
	}
}

func (g *Game) handleConnection(conn net.Conn) {
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	g.mu.Lock()
	err := encoder.Encode(struct {
		Seed   int64
		Tiles  [][]color.RGBA
		Cities [][]bool
	}{
		Seed:   g.seed,
		Tiles:  g.colorToRGBA(g.tiles),
		Cities: g.cities,
	})
	g.mu.Unlock()

	if err != nil {
		log.Println("Ошибка отправки мира:", err)
		return
	}

	var player Player
	if err := decoder.Decode(&player); err != nil {
		log.Println("Ошибка получения данных игрока:", err)
		return
	}

	g.mu.Lock()
	g.players[player.ID] = player
	g.mu.Unlock()

	g.broadcastPlayerUpdate(player)

	for {
		var update Player
		if err := decoder.Decode(&update); err != nil {
			log.Println("Клиент отключился:", err)
			g.mu.Lock()
			delete(g.players, update.ID)
			g.mu.Unlock()
			g.broadcastPlayerUpdate(update)
			return
		}

		g.mu.Lock()
		g.players[update.ID] = update
		g.mu.Unlock()

		g.broadcastPlayerUpdate(update)
	}
}

func (g *Game) broadcastPlayerUpdate(player Player) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, p := range g.players {
		if p.ID == player.ID {
			continue
		}
		if g.conn != nil {
			if err := g.encoder.Encode(player); err != nil {
				log.Println("Ошибка рассылки:", err)
			}
		}
	}
}

func (g *Game) connectToServer() {
	fmt.Println("Введите адрес сервера (например: localhost:8080):")
	var address string
	_, err := fmt.Scanln(&address)
	if err != nil {
		log.Fatal("Ошибка ввода:", err)
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	g.conn = conn
	g.encoder = gob.NewEncoder(conn)
	g.decoder = gob.NewDecoder(conn)

	var world struct {
		Seed   int64
		Tiles  [][]color.RGBA
		Cities [][]bool
	}
	if err := g.decoder.Decode(&world); err != nil {
		log.Fatal("Ошибка получения мира:", err)
	}

	g.seed = world.Seed
	g.tiles = g.rgbaToColor(world.Tiles)
	g.cities = world.Cities

	g.perlin = NewPerlin(g.seed)
	g.noiseMap = make([][]float64, len(g.tiles))
	for y := range g.noiseMap {
		g.noiseMap[y] = make([]float64, len(g.tiles[0]))
	}

	g.generateCities()

	if err := g.encoder.Encode(g.me); err != nil {
		log.Fatal("Ошибка отправки данных игрока:", err)
	}

	go g.handleServerUpdates()
}

func (g *Game) handleServerUpdates() {
	for {
		var player Player
		if err := g.decoder.Decode(&player); err != nil {
			log.Println("Соединение с сервером разорвано:", err)
			return
		}

		g.mu.Lock()
		if player.ID != g.me.ID {
			g.players[player.ID] = player
		}
		g.mu.Unlock()
	}
}

func (g *Game) colorToRGBA(tiles [][]color.Color) [][]color.RGBA {
	rgba := make([][]color.RGBA, len(tiles))
	for y := range tiles {
		rgba[y] = make([]color.RGBA, len(tiles[y]))
		for x := range tiles[y] {
			r, g, b, a := tiles[y][x].RGBA()
			rgba[y][x] = color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}
		}
	}
	return rgba
}

func (g *Game) rgbaToColor(rgba [][]color.RGBA) [][]color.Color {
	tiles := make([][]color.Color, len(rgba))
	for y := range rgba {
		tiles[y] = make([]color.Color, len(rgba[y]))
		for x := range rgba[y] {
			tiles[y][x] = rgba[y][x]
		}
	}
	return tiles
}

func (g *Game) Update() error {
	g.handleMovementInput()
	g.handleCameraInput()
	g.handleCityGenerationInput()
	g.handleCityWindowToggle()
	g.updateHoverCity()
	g.updateCityMap()
	g.cityWindow.Update()

	if inpututil.IsKeyJustPressed(ebiten.KeyP) { // Обработка нажатия P
		g.toggleCharacterWindow()
	}

	if g.characterWindowOpen {
		g.handleCharacterWindowInput()
	}

	if g.conn != nil {
		g.sendPlayerPosition()
	}

	return nil
}

func (g *Game) handleMovementInput() {
	speed := 1
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.me.Y -= speed
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.me.Y += speed
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.me.X -= speed
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.me.X += speed
	}
	// В методе handleMovementInput в main.go
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		mapX, mapY := (mx+g.cameraX)/cellSize, (my+g.cameraY)/cellSize

		// Проверяем клик по городу
		clickedCity := g.findCityAt(mapX, mapY)
		if clickedCity != nil {
			g.initCityMap(clickedCity) // Инициализируем карту города
		}
	}

	cols := screenWidth / cellSize
	rows := screenHeight / cellSize
	g.me.X = clamp(g.me.X, 0, cols-1)
	g.me.Y = clamp(g.me.Y, 0, rows-1)
}

func (g *Game) handleCameraInput() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.cameraX -= 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.cameraX += 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.cameraY -= 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.cameraY += 5
	}
}

func (g *Game) handleCityGenerationInput() {
	if g.mode == "server" && inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.generateNewCities()
	}
	if g.mode == "server" && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.regenerateWorld()
	}
}

func (g *Game) regenerateWorld() {
	g.seed = time.Now().UnixNano()
	g.generateWorld()
	g.generateCities()
}

func (g *Game) handleCityWindowToggle() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.cityWindow.open = !g.cityWindow.open
	}
}

func (g *Game) updateHoverCity() {
	mouseX, mouseY := ebiten.CursorPosition()
	mapX, mapY := (mouseX+g.cameraX)/cellSize, (mouseY+g.cameraY)/cellSize
	g.hoverCity = g.findCityAt(mapX, mapY)
}

func (g *Game) sendPlayerPosition() {
	if err := g.encoder.Encode(g.me); err != nil {
		log.Println("Ошибка отправки позиции:", err)
	}
}

func (g *Game) generateNewCities() {
	g.seed = time.Now().UnixNano()
	g.generateWorld()
	g.generateCities()
	g.cityWindow.cities = g.cityList
}

func (g *Game) Draw(screen *ebiten.Image) {
	for y := range g.tiles {
		for x := range g.tiles[y] {
			screenX := x*cellSize - g.cameraX
			screenY := y*cellSize - g.cameraY

			if g.isVisible(screenX, screenY, cellSize, cellSize) {
				ebitenutil.DrawRect(
					screen,
					float64(screenX),
					float64(screenY),
					cellSize,
					cellSize,
					g.tiles[y][x],
				)
			}
		}
	}

	if g.characterWindowOpen {
		g.drawCharacterWindow(screen)
	}

	debugInfo := fmt.Sprintf(
		"Городов: %d | Камера: (%d, %d) | Seed: %d",
		len(g.cityList),
		g.cameraX,
		g.cameraY,
		g.seed,
	)
	text.Draw(screen, debugInfo, g.font, 10, screenHeight-30, color.White)

	g.mu.Lock()
	for _, player := range g.players {
		if player.ID == g.me.ID {
			continue
		}
		ebitenutil.DrawRect(
			screen,
			float64(player.X*cellSize),
			float64(player.Y*cellSize),
			cellSize,
			cellSize,
			player.Color,
		)
	}
	g.mu.Unlock()

	ebitenutil.DrawRect(
		screen,
		float64(g.me.X*cellSize),
		float64(g.me.Y*cellSize),
		cellSize,
		cellSize,
		color.RGBA{255, 255, 255, 255},
	)

	info := fmt.Sprintf("Режим: %s | ID: %s\n", g.mode, g.me.ID)
	if g.mode == "server" {
		info += "Нажмите R для новой карты\n"
	}
	info += fmt.Sprintf("Позиция: %d, %d\nИгроков онлайн: %d\nTPS: %0.2f",
		g.me.X, g.me.Y, len(g.players), ebiten.ActualTPS())
	ebitenutil.DebugPrint(screen, info)

	g.drawCities(screen)
	g.cityWindow.Draw(screen)
	g.drawCityMap(screen) // Рисуем карту города поверх всего
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Если открыта карта города, увеличиваем окно
	if g.cityMap != nil && g.cityMap.Open {
		return cityMapSize * buildingSize * cityMapScale,
			cityMapSize*buildingSize*cityMapScale + 40
	}
	return screenWidth, screenHeight
}

func (g *Game) isVisible(x, y, width, height int) bool {
	return x+width > 0 && x < screenWidth &&
		y+height > 0 && y < screenHeight
}

func initConsole() {
	// Для Windows пытаемся настроить UTF-8
	if isWindows() {
		exec.Command("chcp", "65001").Run()
	}
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
