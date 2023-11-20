package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"

	"image"

	"math/rand"
	"strings"
	"time"

	_ "image"
	"log"
	_ "math/rand"
	"path"
	_ "time"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	WINDOW_WIDTH         = 1000
	WINDOW_HEIGHT        = 1000
	PLAYERS_HEIGHT       = 64
	PLAYERS_WIDTH        = 64
	NPC1_HEIGHT          = 72
	NPC1_WIDTH           = 64
	FRAMES_PER_SHEET     = 8
	NPC_FRAMES_PER_SHEET = 3
	FRAMES_COUNT         = 4
	numberOfShootNpcs    = 4
	numberOfRegNpcs      = 3
)
const (
	UP = iota
	LEFT
	DOWN
	RIGHT
)

type game struct {
	curMap         *tiled.Map
	mapIterator    int
	tileDict       map[uint32]*ebiten.Image
	boundTiles     []boundaries
	mainplayer     player
	shootnpc       []player
	regnpc         []player
	pathFindingMap []string
	pathMap        *paths.Grid
	path           *paths.Path
	playershots    []Shot
	enemyshots     []Shot
	spawnrate      int
	score          int
	fires          []obj
	chosenNum      int
	gameOver       bool
}

type player struct {
	spriteSheet *ebiten.Image
	xLoc        int
	yLoc        int
	direction   int
	pframe      int
	pframeDelay int
	health      int
	typing      string
	chosen      bool
}

type boundaries struct {
	boundTileX  float64
	boundTileY  float64
	boundWidth  float64
	boundHeight float64
}

type Shot struct {
	pict   *ebiten.Image
	xShot  int
	yShot  int
	deltaX int
	typing string
}

type obj struct {
	pict *ebiten.Image
	xObj int
	yObj int
}

func (game *game) Update() error {
	getPlayerInput(game)
	checkPlayerCollisions(game)
	checkEnemyCollisions(game, game.shootnpc)
	checkEnemyCollisions(game, game.regnpc)
	checkShotCollisions(game, game.playershots)
	checkShotCollisions(game, game.enemyshots)
	checkChosen(game)
	headToPlayer(game)
	NpcAnimation(game, game.shootnpc)
	NpcAnimation(game, game.regnpc)

	game.mainplayer.pframeDelay += 1
	X, Y := game.mainplayer.xLoc, game.mainplayer.yLoc
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && X > 0 {
		game.mainplayer.xLoc -= 1
		if checkPlayerCollisions(game) {
			game.mainplayer.xLoc += 3
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && X < WINDOW_WIDTH-PLAYERS_WIDTH {
		game.mainplayer.xLoc += 1
		if checkPlayerCollisions(game) {
			game.mainplayer.xLoc -= 3
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && Y > 0 {
		game.mainplayer.yLoc -= 1
		if checkPlayerCollisions(game) {
			game.mainplayer.yLoc += 3
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && Y < WINDOW_HEIGHT-PLAYERS_HEIGHT {
		game.mainplayer.yLoc += 1
		if checkPlayerCollisions(game) {
			game.mainplayer.yLoc -= 3
		}
	}

	if game.mainplayer.pframeDelay%FRAMES_COUNT == 0 {
		game.mainplayer.pframe += 1
		if game.mainplayer.pframe >= FRAMES_PER_SHEET {
			game.mainplayer.pframe = 0

		}
	}

	return nil
}

func (game *game) Draw(screen *ebiten.Image) {
	// Drawing the map tiles
	drawOptions := ebiten.DrawImageOptions{}
	for tileY := 0; tileY < game.curMap.Height; tileY += 1 {
		for tileX := 0; tileX < game.curMap.Width; tileX += 1 {
			drawOptions.GeoM.Reset()
			TileXpos := float64(game.curMap.TileWidth * tileX)
			TileYpos := float64(game.curMap.TileHeight * tileY)
			drawOptions.GeoM.Translate(TileXpos, TileYpos)
			tileToDraw := game.curMap.Layers[0].Tiles[tileY*game.curMap.Width+tileX]
			ebitenTileToDraw := game.tileDict[tileToDraw.ID]
			screen.DrawImage(ebitenTileToDraw, &drawOptions)
		}
	}

	// Draw Player
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(float64(game.mainplayer.xLoc), float64(game.mainplayer.yLoc))
	screen.DrawImage(game.mainplayer.spriteSheet.SubImage(image.Rect(
		game.mainplayer.pframe*PLAYERS_WIDTH,
		game.mainplayer.direction*PLAYERS_HEIGHT,
		(game.mainplayer.pframe)*PLAYERS_WIDTH+PLAYERS_WIDTH,
		(game.mainplayer.direction)*PLAYERS_HEIGHT+PLAYERS_HEIGHT)).(*ebiten.Image), &drawOptions)

	for _, npc := range game.regnpc {
		drawOptions := ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(float64(npc.xLoc), float64(npc.yLoc))
		screen.DrawImage(npc.spriteSheet.SubImage(image.Rect(
			npc.pframe*NPC1_WIDTH,
			npc.direction*NPC1_HEIGHT,
			(npc.pframe+1)*NPC1_WIDTH,
			(npc.direction+1)*NPC1_HEIGHT)).(*ebiten.Image), &drawOptions)
	}

	// Draw shooting NPCs
	for _, npc := range game.shootnpc {
		drawOptions := ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(float64(npc.xLoc), float64(npc.yLoc))
		screen.DrawImage(npc.spriteSheet.SubImage(image.Rect(
			npc.pframe*NPC1_WIDTH,
			npc.direction*NPC1_HEIGHT,
			(npc.pframe+1)*NPC1_WIDTH,
			(npc.direction+1)*NPC1_HEIGHT)).(*ebiten.Image), &drawOptions)
	}
	if game.gameOver {
		// Display Game Over message
		DrawCenteredText(screen, "Game Over", WINDOW_WIDTH/2, WINDOW_HEIGHT/2, game)
		return
	}

	//draw text
	DrawCenteredText(screen, fmt.Sprintf("Score: %d", game.score), 100, 12, game)
	DrawCenteredText(screen, fmt.Sprintf("Health: %d", game.mainplayer.health), 250, 12, game)
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "map1.tmx"))
	pathMap := makeSearchMap(gameMap)
	animationGuy := LoadEmbeddedImage("", "dude.png")
	animationOldLady := LoadEmbeddedImage("", "oldlady.png")
	animationOldMan := LoadEmbeddedImage("", "oldman.png")
	animationWarrior := LoadEmbeddedImage("", "warrior.png")
	animationShooter := LoadEmbeddedImage("", "shooter.png")
	rand.Seed(time.Now().UnixNano())

	regNpcs := []player{
		{spriteSheet: animationOldMan, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2},  // NPC1
		{spriteSheet: animationWarrior, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2}, // NPC2
		{spriteSheet: animationOldLady, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2}, // NPC3
	}
	shootNpcs := []player{
		{spriteSheet: animationShooter, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2}, // NPC4
	}
	getRandomPosition := func(maxWidth, maxHeight, npcWidth, npcHeight int) (int, int) {
		x := rand.Intn(maxWidth - npcWidth)
		y := rand.Intn(maxHeight - npcHeight)
		return x, y
	}

	regNpcs = make([]player, numberOfRegNpcs)
	for i := range regNpcs {
		x, y := getRandomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		regNpcs[i] = player{spriteSheet: animationOldMan, xLoc: x, yLoc: y, typing: "reg"}
	}

	shootNpcs = make([]player, numberOfShootNpcs)
	for i := range shootNpcs {
		x, y := getRandomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		shootNpcs[i] = player{spriteSheet: animationShooter, xLoc: x, yLoc: y, typing: "shoot"}
	}
	myPlayer := player{spriteSheet: animationGuy, xLoc: WINDOW_WIDTH / 2, yLoc: 300, health: 3}
	fmt.Printf("Initial Player Health: %d\n", myPlayer.health)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('3', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Jared Plante and Ronaldo Auguste Project 3")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	game := game{
		curMap:         gameMap,
		tileDict:       ebitenImageMap,
		mainplayer:     myPlayer,
		regnpc:         regNpcs,
		shootnpc:       shootNpcs,
		pathFindingMap: pathMap,
		pathMap:        searchablePathMap,
	}
	createBoundSlice(&game)
	err := ebiten.RunGame(&game)
	if err != nil {
		fmt.Println("Couldn't run game:", err)
	}

}

// util funcs

func NpcAnimation(game *game, npcs []player) {
	for i := 0; i < len(npcs); i++ {
		npcs[i].pframeDelay += 1
		if npcs[i].pframeDelay%6 == 0 {
			npcs[i].pframe += 1
			if npcs[i].pframe >= NPC_FRAMES_PER_SHEET {
				npcs[i].pframe = 0

			}
		}
	}
}

func killEnemy(game *game, npcs []player, iterator int) {
	if npcs[iterator].chosen == true {
		game.chosenNum -= 1
	}
	//shift elements to remove enemies
	npcs = append(npcs[:iterator], npcs[iterator+1:]...)
	iterator--
}

func killShots(game *game, shots []Shot, iterator int) {
	//shift elements to remove projectiles
	shots = append(shots[:iterator], shots[iterator+1:]...)
	iterator--
}

func playerLifeLoss(game *game) {
	game.mainplayer.health -= 1
	fmt.Printf("Player Health: %d\n", game.mainplayer.health)
	if game.mainplayer.health <= 0 {
		handleDeath(game)
	} else {
		// respawn the player at specific location
		game.mainplayer.xLoc = 100
		game.mainplayer.yLoc = 100
	}
}

// maybe add in healthbar?
func handleDeath(game *game) {
	game.gameOver = true
}

//ai

func checkChosen(game *game) {
	if game.chosenNum == 0 {
		curShoot := len(game.shootnpc)
		curReg := len(game.regnpc)
		if curShoot != 0 {
			game.shootnpc[rand.Intn(curShoot)].chosen = true
			game.chosenNum += 1
		}
		if curReg != 0 {
			game.regnpc[rand.Intn(curReg)].chosen = true
			game.chosenNum += 1
		}

	}
}

func headToPlayer(game *game) {
	for i := 0; i < len(game.shootnpc); i++ {
		if game.shootnpc[i].chosen {
			startRow := int(game.shootnpc[i].yLoc) / game.curMap.TileHeight
			startCol := int(game.shootnpc[i].xLoc) / game.curMap.TileWidth
			startCell := game.pathMap.Get(startCol, startRow)
			endCell := game.pathMap.Get(game.mainplayer.xLoc/game.curMap.TileWidth, game.mainplayer.yLoc/game.curMap.TileHeight)
			game.path = game.pathMap.GetPathFromCells(startCell, endCell, false, false)
		}
	}
}

//collisions

func getPlayerBounds(game *game) collision.BoundingBox {
	playerBounds := collision.BoundingBox{
		X:      float64(game.mainplayer.xLoc),
		Y:      float64(game.mainplayer.yLoc),
		Width:  float64(PLAYERS_WIDTH),
		Height: float64(PLAYERS_HEIGHT),
	}
	return playerBounds
}

func getShooterBounds(game *game, iterator int) collision.BoundingBox {
	shooterBounds := collision.BoundingBox{
		X:      float64(game.shootnpc[iterator].xLoc),
		Y:      float64(game.shootnpc[iterator].yLoc),
		Width:  float64(NPC1_WIDTH),
		Height: float64(NPC1_HEIGHT),
	}
	return shooterBounds
}

func getRegBounds(game *game, iterator int) collision.BoundingBox {
	regBounds := collision.BoundingBox{
		X:      float64(game.regnpc[iterator].xLoc),
		Y:      float64(game.regnpc[iterator].yLoc),
		Width:  float64(NPC1_WIDTH),
		Height: float64(NPC1_HEIGHT),
	}
	return regBounds
}

func getPlayerShotBounds(game *game, iterator int) collision.BoundingBox {
	regBounds := collision.BoundingBox{
		X:      float64(game.playershots[iterator].xShot),
		Y:      float64(game.playershots[iterator].yShot),
		Width:  float64(game.playershots[iterator].pict.Bounds().Dx()),
		Height: float64(game.playershots[iterator].pict.Bounds().Dy()),
	}
	return regBounds
}

func getEnemyShotBounds(game *game, iterator int) collision.BoundingBox {
	regBounds := collision.BoundingBox{
		X:      float64(game.enemyshots[iterator].xShot),
		Y:      float64(game.enemyshots[iterator].yShot),
		Width:  float64(game.enemyshots[iterator].pict.Bounds().Dx()),
		Height: float64(game.enemyshots[iterator].pict.Bounds().Dy()),
	}
	return regBounds
}

func getFireBounds(game *game, iterator int) collision.BoundingBox {
	fireBounds := collision.BoundingBox{
		X:      float64(game.fires[iterator].xObj),
		Y:      float64(game.fires[iterator].yObj),
		Width:  float64(game.fires[iterator].pict.Bounds().Dx()),
		Height: float64(game.fires[iterator].pict.Bounds().Dy()),
	}
	return fireBounds
}

func getTileBounds(game *game, iterator int) collision.BoundingBox {
	tileBounds := collision.BoundingBox{
		X:      game.boundTiles[iterator].boundTileX,
		Y:      game.boundTiles[iterator].boundTileY,
		Width:  game.boundTiles[iterator].boundWidth,
		Height: game.boundTiles[iterator].boundHeight,
	}
	return tileBounds
}

// lose life if true
func checkPlayerCollisions(game *game) bool {
	playerBounds := getPlayerBounds(game)
	for i := 0; i < len(game.shootnpc); i++ {
		shooterBounds := getShooterBounds(game, i)
		if collision.AABBCollision(playerBounds, shooterBounds) {
			playerLifeLoss(game)
			return true
		}
	}
	for i := 0; i < len(game.regnpc); i++ {
		regBounds := getRegBounds(game, i)
		if collision.AABBCollision(playerBounds, regBounds) {
			playerLifeLoss(game)
			return true
		}
	}
	for i := 0; i < len(game.enemyshots); i++ {
		enemyShotsBounds := getEnemyShotBounds(game, i)
		if collision.AABBCollision(playerBounds, enemyShotsBounds) {
			playerLifeLoss(game)
			killShots(game, game.enemyshots, i)
			return true
		}
	}
	for i := 0; i < len(game.fires); i++ {
		fireBounds := getFireBounds(game, i)
		if collision.AABBCollision(playerBounds, fireBounds) {
			playerLifeLoss(game)
			return true
		}
	}
	for i := 0; i < len(game.boundTiles); i++ {
		tileBounds := getTileBounds(game, i)
		if collision.AABBCollision(playerBounds, tileBounds) {
			playerLifeLoss(game)
			return true
		}
	}
	return false
}

// enemy dies if true (for both regular and shooting enemies)
func checkEnemyCollisions(game *game, npcs []player) bool {
	for j := 0; j < len(npcs); j++ {
		enemyBounds := collision.BoundingBox{}
		if npcs[j].typing == "shoot" {
			enemyBounds = getShooterBounds(game, j)
		} else if npcs[j].typing == "reg" {
			enemyBounds = getRegBounds(game, j)
		}
		for i := 0; i < len(game.shootnpc); i++ {
			// make sure enemy does not collide with itself
			if j != i || npcs[j].typing != "shoot" {
				shooterBounds := getShooterBounds(game, i)
				if collision.AABBCollision(enemyBounds, shooterBounds) {
					killEnemy(game, npcs, j)
					killEnemy(game, game.shootnpc, i)
					return true
				}
			}
		}
		for i := 0; i < len(game.regnpc); i++ {
			// make sure enemy does not collide with itself
			if j != i || npcs[j].typing != "reg" {
				regBounds := getRegBounds(game, i)
				if collision.AABBCollision(enemyBounds, regBounds) {
					killEnemy(game, npcs, j)
					killEnemy(game, game.regnpc, i)
					return true
				}
			}
		}
		for i := 0; i < len(game.playershots); i++ {
			playerShotsBounds := getPlayerShotBounds(game, i)
			if collision.AABBCollision(enemyBounds, playerShotsBounds) {
				game.score += 1
				killEnemy(game, npcs, j)
				killShots(game, game.playershots, i)
				return true
			}
		}
		for i := 0; i < len(game.fires); i++ {
			fireBounds := getFireBounds(game, i)
			if collision.AABBCollision(enemyBounds, fireBounds) {
				killEnemy(game, npcs, j)
				return true
			}
		}
		for i := 0; i < len(game.boundTiles); i++ {
			tileBounds := getTileBounds(game, i)
			if collision.AABBCollision(enemyBounds, tileBounds) {
				killEnemy(game, npcs, j)
				return true
			}
		}
	}
	return false
}

// shots don't go through boundaries (for both player and enemy shots)
func checkShotCollisions(game *game, shots []Shot) bool {
	for j := 0; j < len(shots); j++ {
		shotBounds := collision.BoundingBox{}
		if shots[j].typing == "player" {
			shotBounds = getPlayerShotBounds(game, j)
		} else if shots[j].typing == "npc" {
			shotBounds = getEnemyShotBounds(game, j)
		}
		for i := 0; i < len(game.playershots); i++ {
			// make sure shot does not collide with itself
			if j != i {
				playerShotBounds := getPlayerShotBounds(game, i)
				if collision.AABBCollision(shotBounds, playerShotBounds) {
					killShots(game, shots, j)
					killShots(game, game.playershots, i)
					return true
				}
			}
		}
		for i := 0; i < len(game.enemyshots); i++ {
			// make sure shot does not collide with itself
			if j != i {
				enemyShotBounds := getEnemyShotBounds(game, i)
				if collision.AABBCollision(shotBounds, enemyShotBounds) {
					killShots(game, shots, j)
					killShots(game, game.enemyshots, i)
					return true
				}
			}
		}
		for i := 0; i < len(game.boundTiles); i++ {
			tileBounds := getTileBounds(game, i)
			if collision.AABBCollision(shotBounds, tileBounds) {
				killShots(game, shots, j)
				return true
			}
		}
	}
	return false
}

//text

func DrawCenteredText(screen *ebiten.Image, s string, cx, cy int, game *game) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(basicfont.Face7x13, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2

	// draw text box
	rectWidth := bounds.Dx() + 10 + game.score
	rectHeight := bounds.Dy() + 5
	ebitenutil.DrawRect(screen, float64(x)-5, float64(y)-13, float64(rectWidth), float64(rectHeight), colornames.Burlywood)
	text.Draw(screen, s, basicfont.Face7x13, x, y, colornames.Black)
}

//maps

func createBoundSlice(game *game) {
	for tileY := 0; tileY < game.curMap.Height; tileY += 1 {
		for tileX := 0; tileX < game.curMap.Width; tileX += 1 {
			TileXpos := float64(game.curMap.TileWidth * tileX)
			TileYpos := float64(game.curMap.TileHeight * tileY)
			tileToDraw := game.curMap.Layers[0].Tiles[tileY*game.curMap.Width+tileX]
			if tileToDraw.ID == 3 {
				newBoundTile := boundaries{
					boundTileX:  float64(TileXpos),
					boundTileY:  float64(TileYpos),
					boundWidth:  float64(game.curMap.TileWidth),
					boundHeight: float64(game.curMap.TileHeight),
				}
				game.boundTiles = append(game.boundTiles, newBoundTile)
			}
		}
	}
}

func makeSearchMap(tiledMap *tiled.Map) []string {
	mapAsStringSlice := make([]string, 0, tiledMap.Height) //each row will be its own string
	row := strings.Builder{}
	for position, tile := range tiledMap.Layers[0].Tiles {
		if position%tiledMap.Width == 0 && position > 0 { // we get the 2d array as an unrolled one-d array
			mapAsStringSlice = append(mapAsStringSlice, row.String())
			row = strings.Builder{}
		}
		row.WriteString(fmt.Sprintf("%d", tile.ID))
	}
	mapAsStringSlice = append(mapAsStringSlice, row.String())
	return mapAsStringSlice
}

func makeEbitenImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
	idToImage := make(map[uint32]*ebiten.Image)
	for _, tile := range tiledMap.Tilesets[0].Tiles {
		embeddedFile, err := EmbeddedAssets.Open(path.Join("assets",
			tile.Image.Source))
		if err != nil {
			log.Fatal("failed to load embedded image ", embeddedFile, err)
		}
		ebitenImageTile, _, err := ebitenutil.NewImageFromReader(embeddedFile)
		if err != nil {
			fmt.Println("Error loading tile image:", tile.Image.Source, err)
		}
		idToImage[tile.ID] = ebitenImageTile
	}
	return idToImage
}

func (m game) Layout(oWidth, oHeight int) (sWidth, sHeight int) {
	return oWidth, oHeight
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name,
		tiled.WithFileSystem(EmbeddedAssets))
	if err != nil {
		fmt.Println("Error loading embedded map:", err)
	}
	return embeddedMap
}

// embed

func LoadEmbeddedImage(folderName string, imageName string) *ebiten.Image {
	embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		log.Fatal("failed to load embedded image ", imageName, err)
	}
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tile image:", imageName, err)
	}
	return ebitenImage
}

func getPlayerInput(game *game) {
	if game.gameOver {
		game.mainplayer.xLoc += 0
		game.mainplayer.yLoc += 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && game.mainplayer.xLoc > 0 {
		game.mainplayer.xLoc -= 5
		game.mainplayer.direction = LEFT
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && game.mainplayer.xLoc < WINDOW_WIDTH-PLAYERS_WIDTH {
		game.mainplayer.xLoc += 5
		game.mainplayer.direction = RIGHT
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && game.mainplayer.yLoc > 0 {
		game.mainplayer.yLoc -= 5
		game.mainplayer.direction = UP
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && game.mainplayer.yLoc < WINDOW_HEIGHT-PLAYERS_HEIGHT {
		game.mainplayer.yLoc += 5
		game.mainplayer.direction = DOWN
	}

}
