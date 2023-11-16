package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
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
	Level          *tiled.Map
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

func (game *game) Update() error {
	return nil

}

func (game *game) Draw(screen *ebiten.Image) {
	// Drawing the map tiles
	drawOptions := ebiten.DrawImageOptions{}
	for tileY := 0; tileY < game.Level.Height; tileY += 1 {
		for tileX := 0; tileX < game.Level.Width; tileX += 1 {
			drawOptions.GeoM.Reset()
			TileXpos := float64(game.Level.TileWidth * tileX)
			TileYpos := float64(game.Level.TileHeight * tileY)
			drawOptions.GeoM.Translate(TileXpos, TileYpos)
			tileToDraw := game.Level.Layers[0].Tiles[tileY*game.Level.Width+tileX]
			ebitenTileToDraw := game.tileDict[tileToDraw.ID]
			screen.DrawImage(ebitenTileToDraw, &drawOptions)
		}
	}
}

func main() {

}

// util funcs

//maps

func makeEbiteImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
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
