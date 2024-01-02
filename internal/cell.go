package internal

import (
	"github.com/google/uuid"
	"sync"
)

type Inventory struct {
	energy int16
}

type Cell struct {
	Inventory
	genomeID        string
	genomePosition  uint8
	cellType        CellType
	direction       Direction
	organismID      string
	age             int16
	flowerTimer     int
	seedFlyingTimer int
}

func (c *Cell) GetType() CellType {
	return c.cellType
}

func (c *Cell) ExecuteGenome(world *World, wg *sync.WaitGroup) {
	defer wg.Done()
	if c.cellType != CellTypeSeed && c.cellType != CellTypeSprout {
		return
	}
	var action Action
	if c.seedFlyingTimer > 0 && c.cellType == CellTypeSeed {
		action = NewActionMove(0)
		c.seedFlyingTimer -= 1
	} else {
		genome := world.GetGenome(c.genomeID)
		action = genome.ExecutePosition(c.genomePosition, world)
	}
	action.Apply(c, world)
}

func (c *Cell) ExecuteTypeAction(w *World, wg *sync.WaitGroup) {
	defer wg.Done()
	switch c.cellType {
	case CellTypeSprout:
	case CellTypeTrunk:
	case CellTypeLeaf:
		c.getSunEnergy(w)
	case CellTypeRoot, CellTypeSeed:
		c.getOrganicEnergy(w)
	case CellTypeFlower:
		c.tryToCreateSeed(w)
	}
}

func (c *Cell) getSunEnergy(w *World) {
	pos := w.cellPositions[c]
	ns := pos.Neighbours()
	for i := range ns {
		if !w.Occupied(ns[i]) {
			continue
		}
		if w.GetCellByPosition(ns[i]).cellType == CellTypeLeaf {
			// neighbouring leaves eliminate each other
			return
		}
	}
	c.energy += w.properties[pos].sunlight
}

func (c *Cell) tryToCreateSeed(w *World) {
	if c.flowerTimer > 0 {
		c.flowerTimer -= 1
		return
	}
	if c.energy <= FlowerSpawnEnergy {
		return
	}
	pos := w.GetPosition(c)
	direction := DirectionMax
	possibleDirections := []Direction{DirectionWest, DirectionEast, DirectionNorth, DirectionSouth}
	flowerDirectionFree := false
	for i := range possibleDirections {
		if !w.Occupied(pos.MovedByDirection(possibleDirections[i])) {
			direction = possibleDirections[i]
			if c.direction == direction {
				flowerDirectionFree = true
			}
		}
	}
	if direction == DirectionMax {
		return
	}

	if flowerDirectionFree {
		direction = c.direction
	}

	c.flowerTimer = int(FlowerSpawnEnergy)
	futureGenome := w.GetGenome(c.genomeID).Copy(w)
	newSeed := NewCell(futureGenome.id, CellTypeSeed, c.energy-FlowerSpawnEnergy, uuid.NewString())
	c.energy = FlowerSpawnEnergy
	newSeed.direction = c.direction
	newSeed.seedFlyingTimer = c.seedFlyingTimer

	newLocation := pos.MovedByDirection(direction)
	w.RegisterNewCell(newSeed, newLocation, futureGenome)
}

func (c *Cell) SpendEnergy(w *World) {
	// TODO: tmp solution
	tax := EnergyTax
	if c.cellType == CellTypeSeed || c.cellType == CellTypeSprout || c.cellType == CellTypeTrunk {
		tax /= 2
	}
	c.energy = min(MaxEnergy, max(0, c.energy-tax))
}

func (c *Cell) Die(w *World) {
	pos := w.cellPositions[c]
	w.properties[pos].organic += transformationEnergy(c.cellType) * rotMultiplier
	delete(w.cellPositions, c)
}

func (c *Cell) TooOld() bool {
	if c.cellType == CellTypeSeed {
		return c.age > MaxAge*3
	}
	return c.age > MaxAge
}

func (c *Cell) CheckEnergy(e int16) bool {
	return c.energy > e
}

func (c *Cell) getOrganicEnergy(w *World) {
	got := w.DrainOrganic(w.cellPositions[c])
	c.energy += got
}

func NewCell(genomeID string, ct CellType, energy int16, organismID string) *Cell {
	return &Cell{cellType: ct, genomeID: genomeID, Inventory: Inventory{energy: energy}, organismID: organismID}
}
