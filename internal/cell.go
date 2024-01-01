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
	genomeID       string
	genomePosition uint8
	cellType       CellType
	direction      Direction
	organismID     string
	age            int16
}

func (c *Cell) GetType() CellType {
	return c.cellType
}

func (c *Cell) ExecuteGenome(world *World, wg *sync.WaitGroup) {
	defer wg.Done()
	if c.cellType != CellTypeSeed && c.cellType != CellTypeSprout {
		return
	}
	genome := world.GetGenome(c.genomeID)
	var action Action
	action = genome.ExecutePosition(c.genomePosition, world)
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
	c.energy = min(256, w.properties[pos].sunlight+c.energy)
}

func (c *Cell) tryToCreateSeed(w *World) {
	if c.energy < SeedEnergy {
		return
	}
	c.energy -= SeedEnergy
	genome := w.GetGenome(c.genomeID)
	childGenome := NewGenome(genome)
	child := NewCell(childGenome.id, CellTypeSeed, SeedEnergy, uuid.NewString())
	child.direction = c.direction
	child.genomeID = childGenome.id
	childPos := w.cellPositions[c].MovedByDirection(c.direction)
	w.RegisterNewCell(child, childPos, childGenome)
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
	w.properties[pos].organic += transformationEnergy(c.cellType) / 2
	delete(w.cellPositions, c)
}

func (c *Cell) TooOld() bool {
	if c.cellType == CellTypeSeed || c.cellType == CellTypeSprout {
		return false
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

type Organism struct {
	cells []*Cell
}

func (o *Organism) RegisterCell(cell *Cell) {
	o.cells = append(o.cells, cell)
}

func (o *Organism) HandleEnergyFlow() {
	if len(o.cells) == 0 {
		return
	}
	totalEnergy := int16(0)
	for ix := range o.cells {
		totalEnergy += o.cells[ix].energy
	}
	spreadEnergy := totalEnergy / int16(len(o.cells))
	for ix := range o.cells {
		o.cells[ix].energy = spreadEnergy
	}
}
