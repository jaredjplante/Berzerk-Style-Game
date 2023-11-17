package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
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
}

type player struct {
	spriteSheet *ebiten.Image
	xLoc        int
	yLoc        int
	direction   int
	pframe      int
	pframeDelay int
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
}

type obj struct {
	pict *ebiten.Image
	xObj int
	yObj int
}

func (game *game) Update() error {
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
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "map1.tmx"))
	pathMap := makeSearchMap(gameMap)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('4', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Maps Embedded")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	oneLevelGame := game{
		curMap:         gameMap,
		tileDict:       ebitenImageMap,
		pathFindingMap: pathMap,
		pathMap:        searchablePathMap,
	}
	err := ebiten.RunGame(&oneLevelGame)
	if err != nil {
		fmt.Println("Couldn't run game:", err)
	}
}

// util funcs

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

func checkPlayerCollisions(game *game) bool {
	playerBounds := getPlayerBounds(game)
	for i := 0; i < len(game.shootnpc); i++ {
		shooterBounds := getShooterBounds(game, i)
		if collision.AABBCollision(playerBounds, shooterBounds) {
			return true
		}
	}
	for i := 0; i < len(game.regnpc); i++ {
		regBounds := getRegBounds(game, i)
		if collision.AABBCollision(playerBounds, regBounds) {
			return true
		}
	}
	for i := 0; i < len(game.enemyshots); i++ {
		enemyShotsBounds := getEnemyShotBounds(game, i)
		if collision.AABBCollision(playerBounds, enemyShotsBounds) {
			return true
		}
	}
	for i := 0; i < len(game.fires); i++ {
		fireBounds := getFireBounds(game, i)
		if collision.AABBCollision(playerBounds, fireBounds) {
			return true
		}
	}
	for i := 0; i < len(game.boundTiles); i++ {
		tileBounds := getTileBounds(game, i)
		if collision.AABBCollision(playerBounds, tileBounds) {
			return true
		}
	}
	return false
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
