package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	_ "golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	_ "golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"strings"
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
	numberOfRegNpcs      = 3
	SHOT_WIDTH           = 100
	SHOT_HEIGHT          = 90
	SOUND_SAMPLE_RATE    = 48000
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
	textFont       font.Face
	win            bool
	enemyDeath     sound
	enemyShot      sound
	lvlComplete    sound
	lifeLoss       sound
	loseWav        sound
	winWav         sound
	playerShot     sound
	shotCollide    sound
}
type sound struct {
	audioContext *audio.Context
	soundPlayer  *audio.Player
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
	shotWait    int
}

type boundaries struct {
	boundTileX  float64
	boundTileY  float64
	boundWidth  float64
	boundHeight float64
}

type Shot struct {
	pict        *ebiten.Image
	xShot       float64
	yShot       float64
	deltaX      float64
	deltaY      float64
	typing      string
	direction   int
	speed       float64
	rframe      int
	rframeDelay int
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
	game.mapTransition()
	//walkPath(game, game.shootnpc, game.path)
	//walkPath(game, game.regnpc, game.path2)
	NpcAnimation(game, game.shootnpc)
	NpcAnimation(game, game.regnpc)
	//print(game.chosenNum)

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

		//handle shot animation
		for i := range game.playershots {
			// Update the position based on the direction
			game.playershots[i].rframeDelay += 1
			if game.playershots[i].rframeDelay%2 == 0 {
				game.playershots[i].rframe += 1
				switch game.playershots[i].direction {
				case UP:
					game.playershots[i].yShot -= game.playershots[i].speed
					if game.playershots[i].rframe == 3 {
						game.playershots[i].rframe = 0
					}
				case DOWN:
					game.playershots[i].yShot += game.playershots[i].speed
					if game.playershots[i].rframe == 3 {
						game.playershots[i].rframe = 0
					}
				case LEFT:
					game.playershots[i].xShot -= game.playershots[i].speed
					if game.playershots[i].rframe == 3 {
						game.playershots[i].rframe = 0
					}
				case RIGHT:
					game.playershots[i].xShot += game.playershots[i].speed
					if game.playershots[i].rframe == 3 {
						game.playershots[i].rframe = 0
					}
				}
			}
		}
		//handle gameover
		if game.gameOver {
			return nil
		} else {
			if game.win {
				return nil
			}
		}
	}

	walkPath(game, game.shootnpc, game.path)
	walkPath(game, game.regnpc, game.path2)
	npcShots(game)
	updateEnemyShots(game)
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
		DrawLossScreen(screen, game.textFont)
		//DrawCenteredText(screen, fmt.Sprintf("Score: %d", game.score), WINDOW_HEIGHT/2, WINDOW_WIDTH/4, game)
		//DrawCenteredText(screen.game.textFont, game.score)
		return
	}
	if game.win {
		DrawWinScreen(screen, game.textFont)
		ebiten.SetMaxTPS(0)
		return
	}
	for _, shot := range game.playershots {
		drawOptions := ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(shot.xShot, shot.yShot)
		screen.DrawImage(shot.pict.SubImage(image.Rect(
			shot.rframe*SHOT_WIDTH,
			shot.direction*SHOT_HEIGHT,
			(shot.rframe+1)*SHOT_WIDTH,
			(shot.direction+1)*SHOT_HEIGHT)).(*ebiten.Image), &drawOptions)
	}

	for _, shot := range game.enemyshots {
		drawOptions := ebiten.DrawImageOptions{}
		drawOptions.GeoM.Translate(shot.xShot, shot.yShot)
		screen.DrawImage(shot.pict.SubImage(image.Rect(
			shot.rframe*SHOT_WIDTH,
			shot.direction*SHOT_HEIGHT,
			(shot.rframe+1)*SHOT_WIDTH,
			(shot.direction+1)*SHOT_HEIGHT)).(*ebiten.Image), &drawOptions)
	}

	//draw text
	DrawCenteredText2(screen, fmt.Sprintf("Score: %d", game.score), 100, 12, game)
	DrawCenteredText2(screen, fmt.Sprintf("Health: %d", game.mainplayer.health), 250, 12, game)
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "map1.tmx"))
	pathMap := makeSearchMap(gameMap)
	animationGuy := LoadEmbeddedImage("", "dude.png")
	animationOldLady := LoadEmbeddedImage("", "oldlady.png")
	animationOldMan := LoadEmbeddedImage("", "oldman.png")
	animationWarrior := LoadEmbeddedImage("", "warrior.png")
	animationShooter := LoadEmbeddedImage("", "shooter.png")
	customFont := LoadScoreFont()

	time.Now().UnixNano()

	regNpcs := []player{
		{spriteSheet: animationOldMan, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg", chosen: false},  // NPC1
		{spriteSheet: animationWarrior, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg", chosen: false}, // NPC2
		{spriteSheet: animationOldLady, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "reg", chosen: false}, // NPC3
	}
	shootNpcs := []player{
		{spriteSheet: animationShooter, xLoc: WINDOW_WIDTH / 2, yLoc: WINDOW_HEIGHT / 2, typing: "shoot", chosen: false}, // NPC4
	}

	regNpcs = make([]player, 0, numberOfRegNpcs)
	myPlayer := player{spriteSheet: animationGuy, xLoc: 100, yLoc: 100, health: 3}

	fmt.Printf("Initial Player Health: %d\n", myPlayer.health)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('3', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Jared Plante and Ronaldo Auguste Project 3")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)

	//create sound
	soundContext := audio.NewContext(SOUND_SAMPLE_RATE)
	enemyDeath := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "enemydeath.wav", soundContext),
	}
	enemyShot := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "enemyshot.wav", soundContext),
	}
	levelcomplete := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "levelcomplete.wav", soundContext),
	}
	lifeloss := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "lifeloss.wav", soundContext),
	}
	lose := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "lose.wav", soundContext),
	}
	playershot := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "playershot.wav", soundContext),
	}
	shotcollide := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "shotcollide.wav", soundContext),
	}
	win := sound{
		audioContext: soundContext,
		soundPlayer:  LoadEmbeddedSound("", "win.wav", soundContext),
	}
	game := game{
		curMap:         gameMap,
		tileDict:       ebitenImageMap,
		mainplayer:     myPlayer,
		textFont:       customFont,
		regnpc:         regNpcs,
		shootnpc:       shootNpcs,
		pathFindingMap: pathMap,
		pathMap:        searchablePathMap,
		pathMap2:       searchablePathMap,
		//sounds
		enemyDeath:  enemyDeath,
		enemyShot:   enemyShot,
		lvlComplete: levelcomplete,
		lifeLoss:    lifeloss,
		loseWav:     lose,
		playerShot:  playershot,
		shotCollide: shotcollide,
		winWav:      win,
	}
	createBoundSlice(&game)
	randomEnemy(&game)
	err := ebiten.RunGame(&game)
	if err != nil {
		fmt.Println("Couldn't run game:", err)
	}

}

// util funcs

// add shots

func npcShots(game *game) {
	for i := 0; i < len(game.shootnpc); i++ {
		game.shootnpc[i].shotWait += 1
		if game.shootnpc[i].shotWait%100 == 0 {
			shotImg := LoadEmbeddedImage("", "red.png")
			projectile := Shot{
				pict:      shotImg,
				xShot:     float64(game.shootnpc[i].xLoc),
				yShot:     float64(game.shootnpc[i].yLoc),
				direction: game.shootnpc[i].direction,
				typing:    "npc",
				speed:     30, // set the speed of the projectile
			}
			game.enemyshots = append(game.enemyshots, projectile)
			game.enemyShot.soundPlayer.Rewind()
			game.enemyShot.soundPlayer.Play()
		}
	}
}

// shots direction/ speed
func updateEnemyShots(game *game) {
	for i := range game.enemyshots {
		// Update the position based on the direction
		game.enemyshots[i].rframeDelay += 1
		if game.enemyshots[i].rframeDelay%10 == 0 {
			game.enemyshots[i].rframe += 1
			switch game.enemyshots[i].direction {
			case OLDUP:
				game.enemyshots[i].yShot -= game.enemyshots[i].speed
				if game.enemyshots[i].rframe == 3 {
					game.enemyshots[i].rframe = 0
				}
			case OLDDOWN:
				game.enemyshots[i].yShot += game.enemyshots[i].speed
				if game.enemyshots[i].rframe == 3 {
					game.enemyshots[i].rframe = 0
				}
			case OLDLEFT:
				game.enemyshots[i].xShot -= game.enemyshots[i].speed
				if game.enemyshots[i].rframe == 3 {
					game.enemyshots[i].rframe = 0
				}
			case OLDRIGHT:
				game.enemyshots[i].xShot += game.enemyshots[i].speed
				if game.enemyshots[i].rframe == 3 {
					game.enemyshots[i].rframe = 0
				}
			}
		}
	}
}

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
	game.enemyDeath.soundPlayer.Rewind()
	game.enemyDeath.soundPlayer.Play()
	//shift elements to remove enemies
	npcs = append(npcs[:iterator], npcs[iterator+1:]...)
	return npcs
}

func killShots(game *game, shots []Shot, iterator int) []Shot {
	game.shotCollide.soundPlayer.Rewind()
	game.shotCollide.soundPlayer.Play()
	//shift elements to remove projectiles
	shots = append(shots[:iterator], shots[iterator+1:]...)
	return shots
}

func playerLifeLoss(game *game) {
	game.lifeLoss.soundPlayer.Rewind()
	game.lifeLoss.soundPlayer.Play()
	game.mainplayer.health -= 1
	fmt.Printf("Player Health: %d\n", game.mainplayer.health)
	if game.mainplayer.health <= 0 {
		game.loseWav.soundPlayer.Rewind()
		game.loseWav.soundPlayer.Play()
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
	ebiten.SetMaxTPS(0)
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
	//fmt.Println(npc)
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
				npc[i].direction = LEFT
			} else if pathCell.X*game.curMap.TileWidth < int(npc[i].xLoc) {
				direction = -1.0
				npc[i].direction = RIGHT
			}
			Ydirection := 0.0
			if pathCell.Y*game.curMap.TileHeight > int(npc[i].yLoc) {
				Ydirection = 1.0
				npc[i].direction = DOWN
			} else if pathCell.Y*game.curMap.TileHeight < int(npc[i].yLoc) {
				Ydirection = -1.0
				npc[i].direction = UP
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

func getRandomBounds(game *game, x int, y int) collision.BoundingBox {
	randBounds := collision.BoundingBox{
		X:      float64(x),
		Y:      float64(y),
		Width:  float64(NPC1_WIDTH),
		Height: float64(NPC1_HEIGHT),
	}
	return randBounds
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
		Width:  float64(100),
		Height: float64(90),
	}
	return regBounds
}

func getEnemyShotBounds(game *game, iterator int) collision.BoundingBox {
	regBounds := collision.BoundingBox{
		X:      float64(game.enemyshots[iterator].xShot),
		Y:      float64(game.enemyshots[iterator].yShot),
		Width:  float64(100),
		Height: float64(90),
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

func checkSpawnCollisions(game *game, x int, y int) bool {
	randomBounds := getRandomBounds(game, x, y)
	if collision.AABBCollision(randomBounds, getPlayerBounds(game)) {
		return true
	}
	for i := 0; i < len(game.shootnpc); i++ {
		shooterBounds := getShooterBounds(game, i)
		if collision.AABBCollision(randomBounds, shooterBounds) {
			return true
		}

	}
	for i := 0; i < len(game.regnpc); i++ {
		regBounds := getRegBounds(game, i)
		if collision.AABBCollision(randomBounds, regBounds) {
			return true
		}

	}
	for i := 0; i < len(game.fires); i++ {
		fireBounds := getFireBounds(game, i)
		if collision.AABBCollision(randomBounds, fireBounds) {
			return true
		}
	}
	for i := 0; i < len(game.boundTiles); i++ {
		tileBounds := getTileBounds(game, i)
		if collision.AABBCollision(randomBounds, tileBounds) {
			return true
		}
	}
	return false
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

func DrawCenteredText(screen *ebiten.Image, font font.Face, s string, cx, cy int) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(font, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2
	text.Draw(screen, s, font, x, y, colornames.White)
}

//maps

func createBoundSlice(game *game) {
	for tileY := 0; tileY < game.curMap.Height; tileY += 1 {
		for tileX := 0; tileX < game.curMap.Width; tileX += 1 {
			TileXpos := float64(game.curMap.TileWidth * tileX)
			TileYpos := float64(game.curMap.TileHeight * tileY)
			tileToDraw := game.curMap.Layers[0].Tiles[tileY*game.curMap.Width+tileX]
			if tileToDraw.ID == 3 || tileToDraw.ID == 7 || tileToDraw.ID == 16 || tileToDraw.ID == 15 {
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

func LoadEmbeddedSound(folderName string, soundName string, context *audio.Context) *audio.Player {
	embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", folderName, soundName))
	if err != nil {
		log.Fatal("failed to load embedded sound ", soundName, err)
	}
	Sound, err := wav.DecodeWithoutResampling(embeddedFile)
	if err != nil {
		fmt.Println("Error loading sound file:", soundName, err)
	}
	Player, err := context.NewPlayer(Sound)
	if err != nil {
		fmt.Println("Couldn't create sound player: ", err)
	}
	return Player
}

func getPlayerInput(game *game) {
	if game.gameOver {
		game.mainplayer.pframe += 0
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
		shotImg := LoadEmbeddedImage("", "blue.png")
		projectile := Shot{
			pict:      shotImg,
			xShot:     float64(game.mainplayer.xLoc),
			yShot:     float64(game.mainplayer.yLoc),
			direction: game.mainplayer.direction,
			typing:    "player",
			speed:     30, // set the speed of the projectile
		}
		game.playershots = append(game.playershots, projectile)
		game.playerShot.soundPlayer.Rewind()
		game.playerShot.soundPlayer.Play()
	}
}

// loads next map
func (game *game) loadNextMap() {
	fmt.Println("Attempting to load next map...")
	//spawn player at certain location on the new map
	// increment the map number and determine the next map
	game.currMapnumber++
	if game.currMapnumber > 2 {
		fmt.Println("No more maps to load.")
		game.win = true
		game.winWav.soundPlayer.Rewind()
		game.winWav.soundPlayer.Play()
	}
	game.lvlComplete.soundPlayer.Rewind()
	game.lvlComplete.soundPlayer.Play()

	game.mainplayer.xLoc = 100
	game.mainplayer.yLoc = 100
	game.playershots = []Shot{}
	game.enemyshots = []Shot{}
	// spawn enemies for the new map
	randomEnemy(game)

	var nextMapName string
	switch game.currMapnumber {
	//case 1:
	//	nextMapName = "map1.tmx" // incase we ever need to get back to map1
	case 1:
		nextMapName = "map2.tmx"
	case 2:
		nextMapName = "map3.tmx"
	default:
		fmt.Println("No more maps to load.")
		return
	}

	// load the new map
	newMap := loadMapFromEmbedded(path.Join("assets", nextMapName))
	if newMap == nil {
		fmt.Printf("Failed to load %s\n", nextMapName)
		return
	}
	game.curMap = newMap

	// update tileDict for the new map
	game.tileDict = makeEbitenImagesFromMap(*newMap)

	// clears and update boundTiles for the new map
	game.boundTiles = []boundaries{} // clears existing boundaries for new map
	createBoundSlice(game)           // create new boundaries for new map

	fmt.Printf("Map transitioned to %s\n", nextMapName)
}
func randomEnemy(game *game) {
	// clear existing NPCs
	game.shootnpc = []player{}
	game.regnpc = []player{}
	if game.currMapnumber > 3 {
		return
	}

	//  number of enemies based on the current map number
	numRegNpcs := game.currMapnumber + 2   // 2 npcs for map 1, 3 npcs  for map 2, 4 npcs for map 3
	numShootNpcs := game.currMapnumber + 1 // same thing as regnpc

	// generate new regular NPCs
	for i := 0; i < numRegNpcs; i++ {
		x, y := randomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		for checkSpawnCollisions(game, x, y) {
			x, y = randomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		}
		var npc player
		switch i % 3 {
		case 0:
			npc = player{spriteSheet: LoadEmbeddedImage("", "oldman.png"), xLoc: x, yLoc: y, typing: "reg", chosen: false}
		case 1:
			npc = player{spriteSheet: LoadEmbeddedImage("", "warrior.png"), xLoc: x, yLoc: y, typing: "reg", chosen: false}
		case 2:
			npc = player{spriteSheet: LoadEmbeddedImage("", "oldlady.png"), xLoc: x, yLoc: y, typing: "reg", chosen: false}
		}
		game.regnpc = append(game.regnpc, npc)
	}

	// generate new shooting NPCs
	for i := 0; i < numShootNpcs; i++ {
		x, y := randomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		for checkSpawnCollisions(game, x, y) {
			x, y = randomPosition(WINDOW_WIDTH, WINDOW_HEIGHT, NPC1_WIDTH, NPC1_HEIGHT)
		}
		npc := player{spriteSheet: LoadEmbeddedImage("", "shooter.png"), xLoc: x, yLoc: y, typing: "shoot"}
		game.shootnpc = append(game.shootnpc, npc)
	}
}

// spawns enemies in random positions
func randomPosition(maxWidth, maxHeight, npcWidth, npcHeight int) (int, int) {
	//logic for random npc spawning
	x := rand.Intn(maxWidth - NPC1_WIDTH)
	y := rand.Intn(maxHeight - NPC1_HEIGHT)
	return x, y
}

// logic to check if game can move to next map
func (game *game) mapTransition() {
	//if no shoot and regnpcs are alive it will load the next map
	if len(game.shootnpc) == 0 && len(game.regnpc) == 0 {
		if game.currMapnumber < 3 {
			// load the next map
			game.loadNextMap()
		} else {
			game.win = true
		}
	}
}

// loss screen after game ends
func DrawLossScreen(screen *ebiten.Image, font font.Face) {
	screen.Fill(color.Black)
	loseText := "You Lose"
	bounds := text.BoundString(font, loseText)
	x := (WINDOW_WIDTH - bounds.Dx()) / 2
	y := WINDOW_HEIGHT / 2

	text.Draw(screen, loseText, font, x, y, color.White)
}
func DrawWinScreen(screen *ebiten.Image, font font.Face) {
	screen.Fill(color.White) // Fill the screen with black

	winText := "You Win"
	bounds := text.BoundString(font, winText)
	x := (WINDOW_WIDTH - bounds.Dx()) / 2
	y := WINDOW_HEIGHT / 2

	text.Draw(screen, winText, font, x, y, color.Black)
}

func LoadScoreFont() font.Face {
	//originally inspired by https://www.fatoldyeti.com/posts/roguelike16/
	trueTypeFont, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		fmt.Println("Error loading font for score:", err)
	}
	fontFace, err := opentype.NewFace(trueTypeFont, &opentype.FaceOptions{
		Size:    55,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Println("Error loading font of correct size for score:", err)
	}
	return fontFace
}

func DrawCenteredText2(screen *ebiten.Image, s string, cx, cy int, game *game) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(basicfont.Face7x13, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2

	// draw text box
	rectWidth := bounds.Dx() + 10 + game.score
	rectHeight := bounds.Dy() + 5
	ebitenutil.DrawRect(screen, float64(x)-5, float64(y)-13, float64(rectWidth), float64(rectHeight), colornames.Burlywood)
	text.Draw(screen, s, basicfont.Face7x13, x, y, colornames.Black)
}
