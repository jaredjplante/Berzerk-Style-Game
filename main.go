package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
	"image"
	"log"
	"math"
	"strings"

	"math/rand"
	"time"

	_ "image"
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
	FRAMES_COUNT         = 4
	NPC_FRAMES_PER_SHEET = 3
	numberOfShootNpcs    = 4
	numberOfRegNpcs      = 3
)
const (
	UP = iota
	LEFT
	DOWN
	RIGHT
)
const (
	OLDUP = iota
	OLDRIGHT
	OLDDOWN
	OLDLEFT
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
	pathMap2       *paths.Grid
	path2          *paths.Path
	playershots    []Shot
	enemyshots     []Shot
	spawnrate      int
	score          int
	fires          []obj
	chosenNum      int
	gameOver       bool
	currMapnumber  int
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
	pict      *ebiten.Image
	xShot     float64
	yShot     float64
	deltaX    int
	typing    string
	direction int
	speed     float64
}

type obj struct {
	pict *ebiten.Image
	xObj int
	yObj int
}

func (game *game) Update() error {
	getPlayerInput(game)
	checkPlayerCollisions(game)
	game.shootnpc = checkEnemyCollisions(game, game.shootnpc)
	game.regnpc = checkEnemyCollisions(game, game.regnpc)
	game.playershots = checkShotCollisions(game, game.playershots)
	game.enemyshots = checkShotCollisions(game, game.enemyshots)
	checkChosen(game)
	headToPlayer(game)
	//game.checkMapTransition()
	//walkPath(game, game.shootnpc, game.path)
	//walkPath(game, game.regnpc, game.path2)
	NpcAnimation(game, game.shootnpc)
	NpcAnimation(game, game.regnpc)
	print(game.chosenNum)

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
		//game.mainplayer.pframe += 1
		if game.mainplayer.pframe >= FRAMES_PER_SHEET {
			game.mainplayer.pframe = 0

		}
		for i := range game.playershots {
			switch game.playershots[i].direction {
			case UP:
				game.playershots[i].yShot -= game.playershots[i].speed
			case DOWN:
				game.playershots[i].yShot += game.playershots[i].speed
			case LEFT:
				game.playershots[i].xShot -= game.playershots[i].speed
			case RIGHT:
				game.playershots[i].xShot += game.playershots[i].speed
			}
			if game.playershots[i].xShot < 0 || game.playershots[i].xShot > WINDOW_WIDTH ||
				game.playershots[i].yShot < 0 || game.playershots[i].yShot > WINDOW_HEIGHT {

				game.playershots = append(game.playershots[:i], game.playershots[i+1:]...)
				i--
			}
			if game.mainplayer.xLoc == 100 && game.mainplayer.yLoc == 100 {
				// Transition to the next map
				//game.loadNextMap()

			}
			if len(game.shootnpc) == 0 && len(game.regnpc) == 0 {
				//game.loadNextMap()
			}

		}
		if inpututil.IsKeyJustPressed(ebiten.KeyT) {
			//game.loadNextMap()
		}
	}
	walkPath(game, game.shootnpc, game.path)
	walkPath(game, game.regnpc, game.path2)
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
	for _, shot := range game.playershots {
		drawOptions := ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(shot.xShot, shot.yShot)
		screen.DrawImage(shot.pict, &drawOptions)
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
		{spriteSheet: animationOldMan, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg"},  // NPC1
		{spriteSheet: animationWarrior, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg"}, // NPC2
		{spriteSheet: animationOldLady, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg"}, // NPC3
	}
	shootNpcs := []player{
		{spriteSheet: animationShooter, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "shoot"}, // NPC4
	}

	regNpcs = make([]player, 0, numberOfRegNpcs)
	myPlayer := player{spriteSheet: animationGuy, xLoc: 100, yLoc: 100, health: 3}

	fmt.Printf("Initial Player Health: %d\n", myPlayer.health)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('3', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Jared Plante and Ronaldo Auguste Project 3")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	game := game{
		curMap:     gameMap,
		tileDict:   ebitenImageMap,
		mainplayer: myPlayer,

		regnpc:         regNpcs,
		shootnpc:       shootNpcs,
		pathFindingMap: pathMap,
		pathMap:        searchablePathMap,
		pathMap2:       searchablePathMap,
	}
	createBoundSlice(&game)
	randomEnemy(&game)
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
		if npcs[i].pframeDelay%10 == 0 {
			npcs[i].pframe += 1
			if npcs[i].pframe >= NPC_FRAMES_PER_SHEET {
				npcs[i].pframe = 0
			}
			if npcs[i].direction == OLDLEFT {
				npcs[i].xLoc -= 5
			} else if npcs[i].direction == OLDRIGHT {
				npcs[i].xLoc += 5
			} else if npcs[i].direction == OLDUP {
				npcs[i].yLoc -= 5
			} else if npcs[i].direction == OLDDOWN {
				npcs[i].yLoc += 5
			}

		}
	}
}

func killEnemy(game *game, npcs []player, iterator int) []player {
	if npcs[iterator].chosen == true {
		game.chosenNum -= 1
	}
	//shift elements to remove enemies
	npcs = append(npcs[:iterator], npcs[iterator+1:]...)
	return npcs
}

func killShots(game *game, shots []Shot, iterator int) []Shot {
	//shift elements to remove projectiles
	shots = append(shots[:iterator], shots[iterator+1:]...)
	return shots
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
	if game.chosenNum <= 0 {
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
	for i := 0; i < len(game.regnpc); i++ {
		if game.regnpc[i].chosen {
			startRow := int(game.regnpc[i].yLoc) / game.curMap.TileHeight
			startCol := int(game.regnpc[i].xLoc) / game.curMap.TileWidth
			startCell := game.pathMap2.Get(startCol, startRow)
			endCell := game.pathMap2.Get(game.mainplayer.xLoc/game.curMap.TileWidth, game.mainplayer.yLoc/game.curMap.TileHeight)
			game.path2 = game.pathMap2.GetPathFromCells(startCell, endCell, false, false)
		}
	}
}

func walkPath(game *game, npc []player, path *paths.Path) {
	for i := 0; i < len(npc); i++ {
		if path != nil && npc[i].chosen {
			pathCell := path.Current()
			if math.Abs(float64(pathCell.X*game.curMap.TileWidth)-float64(npc[i].xLoc)) <= 2 &&
				math.Abs(float64(pathCell.Y*game.curMap.TileHeight)-float64(npc[i].yLoc)) <= 2 { //if we are now on the tile we need to be on
				path.Advance()
			}
			if path.AtEnd() {
				path = nil
				npc[i].chosen = false
				game.chosenNum -= 1
				return
			}
			direction := 0.0
			if pathCell.X*game.curMap.TileWidth > int(npc[i].xLoc) {
				direction = 1.0
			} else if pathCell.X*game.curMap.TileWidth < int(npc[i].xLoc) {
				direction = -1.0
			}
			Ydirection := 0.0
			if pathCell.Y*game.curMap.TileHeight > int(npc[i].yLoc) {
				Ydirection = 1.0
			} else if pathCell.Y*game.curMap.TileHeight < int(npc[i].yLoc) {
				Ydirection = -1.0
			}
			npc[i].xLoc += int(direction) * 2
			npc[i].yLoc += int(Ydirection) * 2
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
func checkEnemyCollisions(game *game, npcs []player) []player {
	enemyBool := false
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
					game.shootnpc = killEnemy(game, game.shootnpc, i)
					enemyBool = true
				}
			}
		}
		for i := 0; i < len(game.regnpc); i++ {
			// make sure enemy does not collide with itself
			if j != i || npcs[j].typing != "reg" {
				regBounds := getRegBounds(game, i)
				if collision.AABBCollision(enemyBounds, regBounds) {
					game.regnpc = killEnemy(game, game.regnpc, i)
					enemyBool = true
				}
			}
		}
		for i := 0; i < len(game.playershots); i++ {
			playerShotsBounds := getPlayerShotBounds(game, i)
			if collision.AABBCollision(enemyBounds, playerShotsBounds) {
				game.score += 1
				game.playershots = killShots(game, game.playershots, i)
				enemyBool = true
			}
		}
		for i := 0; i < len(game.fires); i++ {
			fireBounds := getFireBounds(game, i)
			if collision.AABBCollision(enemyBounds, fireBounds) {
				enemyBool = true
			}
		}
		for i := 0; i < len(game.boundTiles); i++ {
			tileBounds := getTileBounds(game, i)
			if collision.AABBCollision(enemyBounds, tileBounds) {
				enemyBool = true
			}
		}
		if enemyBool {
			npcs = killEnemy(game, npcs, j)
			enemyBool = false
		}
	}
	return npcs
}

// shots don't go through boundaries (for both player and enemy shots)
func checkShotCollisions(game *game, shots []Shot) []Shot {
	shotHit := false
	for j := 0; j < len(shots); j++ {
		shotBounds := collision.BoundingBox{}
		if shots[j].typing == "player" {
			shotBounds = getPlayerShotBounds(game, j)
		} else if shots[j].typing == "npc" {
			shotBounds = getEnemyShotBounds(game, j)
		}
		for i := 0; i < len(game.playershots); i++ {
			// make sure shot does not collide with itself
			if j != i && shots[j].typing == "npc" {
				playerShotBounds := getPlayerShotBounds(game, i)
				if collision.AABBCollision(shotBounds, playerShotBounds) {
					game.playershots = killShots(game, game.playershots, i)
					shotHit = true
				}
			}
		}
		for i := 0; i < len(game.enemyshots); i++ {
			// make sure shot does not collide with itself
			if j != i && shots[j].typing == "player" {
				enemyShotBounds := getEnemyShotBounds(game, i)
				if collision.AABBCollision(shotBounds, enemyShotBounds) {
					game.enemyshots = killShots(game, game.enemyshots, i)
					shotHit = true
				}
			}
		}
		for i := 0; i < len(game.boundTiles); i++ {
			tileBounds := getTileBounds(game, i)
			if collision.AABBCollision(shotBounds, tileBounds) {
				shotHit = true
			}
		}
		if shotHit {
			shots = killShots(game, shots, j)
			return shots
		}
	}
	return shots
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
		game.mainplayer.pframe += 1
		game.mainplayer.xLoc -= 5
		game.mainplayer.direction = LEFT
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && game.mainplayer.xLoc < WINDOW_WIDTH-PLAYERS_WIDTH {
		game.mainplayer.pframe += 1
		game.mainplayer.xLoc += 5
		game.mainplayer.direction = RIGHT
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && game.mainplayer.yLoc > 0 {
		game.mainplayer.pframe += 1
		game.mainplayer.yLoc -= 5
		game.mainplayer.direction = UP
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && game.mainplayer.yLoc < WINDOW_HEIGHT-PLAYERS_HEIGHT {
		game.mainplayer.pframe += 1
		game.mainplayer.yLoc += 5
		game.mainplayer.direction = DOWN
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		shotImg := LoadEmbeddedImage("", "projectile.png")
		projectile := Shot{
			pict:      shotImg,
			xShot:     float64(game.mainplayer.xLoc),
			yShot:     float64(game.mainplayer.yLoc),
			direction: game.mainplayer.direction,
			typing:    "player",
			speed:     10, // set the speed of the projectile
		}
		game.playershots = append(game.playershots, projectile)
	}
}

//	func (game *game) loadNextMap() {
//		fmt.Println("Attempting to load next map...")
//		randomEnemy(game)
//		game.mainplayer.xLoc = 100
//		game.mainplayer.yLoc = 100
//
//		// Update tile collisions for the new map
//		createBoundSlice(game)
//		if game.currMapnumber == 3 {
//			fmt.Println("No more maps to load.")
//			return
//		}
//		// Increment the map number
//		game.currMapnumber++
//
//		// Determine the next map to load based on the current map number
//		var nextMapName string
//		switch game.currMapnumber {
//		case 2:
//			nextMapName = "map2.tmx"
//		case 3:
//			nextMapName = "map3.tmx"
//		default:
//			fmt.Println("No more maps to load.")
//			return
//		}
//
//		// Load the map and check for errors
//		newMap := loadMapFromEmbedded(path.Join("assets", nextMapName))
//		if newMap == nil {
//			fmt.Printf("Failed to load %s\n", nextMapName)
//			return
//		}
//
//		game.curMap = newMap
//		fmt.Printf("Map transitioned to %s\n", nextMapName)
//
//		// Reset or initialize game state as needed
//		// ...
//	}
func randomEnemy(game *game) {
	// clear existing NPCs
	game.shootnpc = []player{}
	game.regnpc = []player{}

	// generate new NPCs based on the current map
	for i := 0; i < numberOfRegNpcs; i++ {
		x, y := getRandomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		var npc player
		switch i % 1 {
		case 0:
			npc = player{spriteSheet: LoadEmbeddedImage("", "oldman.png"), xLoc: x, yLoc: y, typing: "reg"}
		case 1:
			npc = player{spriteSheet: LoadEmbeddedImage("", "warrior.png"), xLoc: x, yLoc: y, typing: "reg"}
		case 2:
			npc = player{spriteSheet: LoadEmbeddedImage("", "oldlady.png"), xLoc: x, yLoc: y, typing: "reg"}
		}
		game.regnpc = append(game.regnpc, npc)
	}

	for i := 0; i < numberOfShootNpcs; i++ {
		x, y := getRandomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		npc := player{spriteSheet: LoadEmbeddedImage("", "shooter.png"), xLoc: x, yLoc: y, typing: "shoot"}
		game.shootnpc = append(game.shootnpc, npc)
	}
}

func getRandomPosition(maxWidth, maxHeight, npcWidth, npcHeight int) (int, int) {
	x := rand.Intn(maxWidth - NPC1_WIDTH)
	y := rand.Intn(maxHeight - NPC1_HEIGHT)
	return x, y
}

//func (game *game) checkMapTransition() {
//	// check specific conditions for transitioning to the next map
//	if len(game.shootnpc) == 0 && len(game.regnpc) == 0 {
//		game.loadNextMap()
//	}
//}
