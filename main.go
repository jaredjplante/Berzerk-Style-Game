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
	"math/rand"
	"strings"

	_ "image"
	"log"
	"path"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	WINDOW_WIDTH   = 1000
	WINDOW_HEIGHT  = 1000
	PLAYERS_HEIGHT = 112
	PLAYERS_WIDTH  = 100
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
	checkPlayerCollisions(game)
	checkEnemyCollisions(game, game.shootnpc)
	checkEnemyCollisions(game, game.regnpc)
	checkShotCollisions(game, game.playershots)
	checkShotCollisions(game, game.enemyshots)
	checkChosen(game)
	headToPlayer(game)
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
	//draw text
	DrawCenteredText(screen, fmt.Sprintf("Score: %d", game.score), 100, 12, game)
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "map1.tmx"))
	pathMap := makeSearchMap(gameMap)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('4', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Jared Plante and Ronaldo Auguste Project 3")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	game := game{
		curMap:         gameMap,
		tileDict:       ebitenImageMap,
		pathFindingMap: pathMap,
		pathMap:        searchablePathMap,
	}
	err := ebiten.RunGame(&game)
	if err != nil {
		fmt.Println("Couldn't run game:", err)
	}
}

// util funcs

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
	//restart player loc
	//Ronaldo to do
	game.mainplayer.health -= 1
}

func handleDeath(game *game) {
	//game over
	//health reaches 0
	//Ronaldo to do
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
		Width:  float64(PLAYERS_WIDTH),
		Height: float64(PLAYERS_HEIGHT),
	}
	return shooterBounds
}

func getRegBounds(game *game, iterator int) collision.BoundingBox {
	regBounds := collision.BoundingBox{
		X:      float64(game.regnpc[iterator].xLoc),
		Y:      float64(game.regnpc[iterator].yLoc),
		Width:  float64(PLAYERS_WIDTH),
		Height: float64(PLAYERS_HEIGHT),
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
