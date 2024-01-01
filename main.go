package main

import (
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/google/uuid"
	"golang.org/x/image/font/basicfont"
	"image"
	"image/color"
	_ "image/png"
	"math/rand"
	"multicell/internal"
	"os"
)

const SimulationSteps = 10000

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
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
	world.MoveCells()
	world.CreateNewCells()
	world.SpreadEnergy()
	world.DrainEnergy()
	world.RemoveCells()
}

func runSimulation(world *internal.World, exporter chan internal.WorldExport) {
	for i := 0; i < SimulationSteps; i++ {
		runSimulationStep(world)
		exporter <- world.Export()

		if len(world.GetCells()) == 0 {
			seedWorld(world)
		}
	}
	close(exporter)
}

type Resources struct {
	flower, trunk, seed, sprout, leaf, root *pixel.Sprite
}

func loadSprite(path string) *pixel.Sprite {
	pic, err := loadPicture(path)
	if err != nil {
		panic(err)
	}
	return pixel.NewSprite(pic, pic.Bounds())
}

func loadResources() *Resources {
	result := Resources{
		flower: loadSprite("resources/sprites/flower.png"),
		sprout: loadSprite("resources/sprites/sprout.png"),
		trunk:  loadSprite("resources/sprites/trunk.png"),
		seed:   loadSprite("resources/sprites/seed.png"),
		leaf:   loadSprite("resources/sprites/leaf.png"),
		root:   loadSprite("resources/sprites/root.png"),
	}
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

	for !win.Closed() {
		win.Clear(color.Black)
		worldExport, more := <-exporter
		if more {
			basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
			basicTxt := text.New(pixel.V(100, 100), basicAtlas)

			_, err = fmt.Fprintf(basicTxt, "%d\n", worldExport.Turn())
			if err != nil {
				panic(err)
			}
			basicTxt.Draw(win, pixel.IM)
			for pos, _ := range worldExport.CellTypes() {
				var sprite *pixel.Sprite
				switch worldExport.CellTypes()[pos] {
				case internal.CellTypeFlower:
					sprite = resources.flower
				case internal.CellTypeSprout:
					sprite = resources.sprout
				case internal.CellTypeTrunk:
					sprite = resources.trunk
				case internal.CellTypeSeed:
					sprite = resources.seed
				case internal.CellTypeLeaf:
					sprite = resources.leaf
				case internal.CellTypeRoot:
					sprite = resources.root
				}
				if sprite == nil {
					panic(fmt.Errorf("the sprite was empty"))
				}
				sprite.DrawColorMask(win, pixel.IM.Moved(win.Bounds().Min.Add(
					pixel.V(50.0, -50.0),
				).Add(pixel.V(float64(pos.X)*32, float64(pos.Y)*32))).Scaled(
					pixel.V(0.0, 0.0), 0.5),
					color.RGBA{R: uint8(127 + 127*worldExport.Energy()[pos]/internal.MaxEnergy)},
				)
			}
		} else {
			return
		}

		win.Update()
	}
}

func seedWorld(w *internal.World) {
	for i := 0; i < internal.WorldSize; i++ {
		for j := 0; j < internal.WorldSize; j++ {
			pos := internal.NewPosition(int64(i), int64(j))
			if rand.Float32() > 0.1 {
				continue
			}
			parentGenome := internal.NewGenome(nil)
			w.AddGenome(parentGenome)
			w.AddCell(
				pos,
				internal.NewCell(parentGenome.GetID(), internal.CellTypeSeed, 255, uuid.NewString()),
			)
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
