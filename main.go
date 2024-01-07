package main

import (
	"embed"
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/google/uuid"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
	"hash/fnv"
	"image"
	_ "image/png"
	"math"
	"math/rand"
	"multicell/internal"
	"time"
)

const SimulationSteps = 1000000

//go:embed resources/*
var resources embed.FS

func loadPicture(path string) (pixel.Picture, error) {
	file, err := resources.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func runSimulationStep(world *internal.World) {
	world.CleanupTurn()
	world.ExecuteCellGenomes()
	world.ExecuteTypeActions()
	world.CreateNewCells()
	world.SpreadEnergy()
	world.DrainResources()
	world.MoveCells()
	world.RemoveCells()
}

func runSimulation(world *internal.World, exporter chan internal.WorldExport) {
	noProgressCounter := 0
	for i := 0; i < SimulationSteps; i++ {
		runSimulationStep(world)
		exporter <- world.Export()

		canProgress := false
		for c := range world.GetCells() {
			if c.GetType() == internal.CellTypeSprout || c.GetType() == internal.CellTypeFlower {
				canProgress = true
			}
		}
		if !canProgress {
			noProgressCounter += 1
		}
		if noProgressCounter >= 200 {
			seedWorld(world)
			noProgressCounter = 0
		}

	}
	close(exporter)
}

type Resources struct {
	spritesheet pixel.Picture

	framesMap map[internal.CellType]int
	frames    []pixel.Rect
}

func loadResources() *Resources {
	result := Resources{}
	var err error
	result.spritesheet, err = loadPicture("resources/sprites/sheet_small.png")
	if err != nil {
		panic(err)
	}
	result.frames = make([]pixel.Rect, 0)
	result.framesMap = make(map[internal.CellType]int)
	for x := result.spritesheet.Bounds().Min.X; x < result.spritesheet.Bounds().Max.X; x += 32 {
		for y := result.spritesheet.Bounds().Min.Y; y < result.spritesheet.Bounds().Max.Y; y += 32 {
			result.frames = append(result.frames, pixel.R(x, y, x+32, y+32))
		}
	}
	result.framesMap[internal.CellTypeFlower] = 0
	result.framesMap[internal.CellTypeLeaf] = 2
	result.framesMap[internal.CellTypeTrunk] = 4
	result.framesMap[internal.CellTypeSeed] = 6
	result.framesMap[internal.CellTypeSprout] = 8
	result.framesMap[internal.CellTypeRoot] = 10
	result.framesMap[internal.CellTypeConnector] = 12
	return &result
}

func run(exporter chan internal.WorldExport) {
	resources := loadResources()
	cfg := pixelgl.WindowConfig{
		Title:  "Multicell",
		Bounds: pixel.R(0, 0, 1280, 1024),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	var (
		camPos       = pixel.ZV.Add(pixel.V(400, 400))
		camSpeed     = 500.0
		camZoom      = 1.0
		camZoomSpeed = 1.2
	)

	last := time.Now()
	pause := true
	oneStep := false
	visionMode := 0
	keySWasReleased, wasReleased := true, true
	var worldExport internal.WorldExport
	batch := pixel.NewBatch(&pixel.TrianglesData{}, resources.spritesheet)
	more := true
	for !win.Closed() {
		if win.JustPressed(pixelgl.KeySpace) && wasReleased {
			pause = !pause
			wasReleased = false
		}
		if win.JustPressed(pixelgl.KeyS) && keySWasReleased {
			keySWasReleased = false
			oneStep = true
		}
		// TODO: at least make constants
		if win.JustPressed(pixelgl.KeyT) {
			// genome view
			visionMode = 4
		}
		if win.JustPressed(pixelgl.KeyR) {
			// genome view
			visionMode = 3
		}
		if win.JustPressed(pixelgl.KeyW) {
			visionMode = 2
		}
		if win.JustPressed(pixelgl.KeyE) {
			visionMode = 1
		}
		if win.JustPressed(pixelgl.KeyQ) {
			visionMode = 0
		}

		if win.JustReleased(pixelgl.KeyS) {
			keySWasReleased = true
		}

		if win.JustReleased(pixelgl.KeySpace) {
			wasReleased = true
		}
		if (!pause || oneStep) && more {
			oneStep = false
			var tmpWE internal.WorldExport
			tmpWE, more = <-exporter
			if more {
				worldExport = tmpWE
			}
		}

		var cells []*pixel.Sprite
		var matrices []pixel.Matrix
		var colors []*pixel.RGBA

		dt := time.Since(last).Seconds()
		last = time.Now()

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)
		// TODO: common sprite handler
		for pos := range worldExport.CellTypes() {
			rectNum := resources.framesMap[worldExport.CellTypes()[pos]]
			if visionMode != 0 {
				rectNum += 1
			}
			rect := resources.frames[rectNum]
			cells = append(cells, pixel.NewSprite(resources.spritesheet, rect))
			matrices = append(
				matrices,
				pixel.IM.Moved(
					pixel.V(float64(pos.X)*32, float64(pos.Y)*32),
				),
			)
			var color *pixel.RGBA
			if visionMode == 1 {
				color = &pixel.RGBA{R: 0.1 + 0.9*float64(worldExport.Energy()[pos])/float64(internal.MaxEnergy)}
			} else if visionMode == 2 || visionMode == 3 {
				source := worldExport.Organisms()[pos]
				if visionMode == 3 {
					source = worldExport.Genomes()[pos]
				}

				hash := fnv.New32a()
				hash.Write([]byte(source))
				hashValue := hash.Sum32()
				minColorValue := 0.3
				rgbDivider := 255 * (1.0 - minColorValue)
				rgba := pixel.RGBA{
					R: minColorValue + float64(hashValue&0xFF)/rgbDivider,
					G: minColorValue + float64((hashValue>>8)&0xFF)/rgbDivider,
					B: minColorValue + float64((hashValue>>16)&0xFF)/rgbDivider,
					A: 255,
				}
				color = &rgba
			} else if visionMode == 4 {
				color = &pixel.RGBA{B: 0.1 + 0.9*float64(worldExport.Water()[pos])/float64(internal.WaterMaxAmount)}
			}
			colors = append(colors, color)
		}

		if win.Pressed(pixelgl.KeyLeft) {
			camPos.X -= camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyRight) {
			camPos.X += camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyDown) {
			camPos.Y -= camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyUp) {
			camPos.Y += camSpeed * dt
		}
		camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y)

		batch.Clear()
		win.Clear(colornames.Black)
		for i, cell := range cells {
			if colors[i] == nil {
				cell.Draw(batch, matrices[i])
			} else {
				cell.DrawColorMask(batch, matrices[i], colors[i])
			}
		}
		batch.Draw(win)
		basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
		basicTxt := text.New(camPos.Add(pixel.V(-300, -300)), basicAtlas)

		_, err = fmt.Fprintf(basicTxt, "%d\n", worldExport.Turn())
		if err != nil {
			panic(err)
		}

		basicTxt.Draw(win, pixel.IM.Scaled(basicTxt.Orig, 4))
		if pause {
			basicTxt = text.New(camPos.Add(pixel.V(-0, -300)), basicAtlas)

			_, err = fmt.Fprintf(basicTxt, "PAUSED\n")
			if err != nil {
				panic(err)
			}
			basicTxt.Draw(win, pixel.IM.Scaled(basicTxt.Orig, 4))
		}

		win.Update()
	}
}

func seedWorld(w *internal.World) {
	for i := 0; i < internal.WorldSize; i++ {
		for j := 0; j < internal.WorldSize; j++ {
			pos := internal.NewPosition(int64(i), int64(j))
			if rand.Float32() > 0.03 {
				continue
			}
			parentGenome := internal.NewGenome(nil)
			w.AddGenome(parentGenome)
			w.AddCell(
				pos,
				internal.NewCell(
					parentGenome.GetID(), internal.CellTypeSeed,
					internal.Inventory{internal.ItemTypeWater: internal.WaterMaxAmount,
						internal.ItemTypeEnergy: internal.MaxEnergy},
					uuid.NewString(),
				),
			)

			//return
		}
	}
}

func main() {
	world := internal.NewWorld(internal.WorldSize)
	seedWorld(world)
	exporter := make(chan internal.WorldExport)
	go runSimulation(world, exporter)
	runUI := func() {
		run(exporter)
	}
	pixelgl.Run(runUI)
}
