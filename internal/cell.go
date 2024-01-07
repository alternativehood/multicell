package internal

import (
	"sync"

	"github.com/google/uuid"
)

type ItemType int

const (
	ItemTypeEnergy ItemType = iota
	ItemTypeWater
	ItemTypeOrganic
	MaxItemType
)

func itemSpreadStep(t ItemType) int16 {
	switch t {
	case ItemTypeEnergy:
		return EnergyTransferAmount
	case ItemTypeWater:
		return WaterTransferAmount
	case ItemTypeOrganic:
		return 0
	}
	panic(t)
}

type Inventory map[ItemType]int16

type CellInventory struct {
	inventory Inventory

	inventoryMx sync.Mutex
}

func (c *CellInventory) GetFromInventory(itemType ItemType) int16 {
	c.inventoryMx.Lock()
	defer c.inventoryMx.Unlock()
	return c.inventory[itemType]
}

func (c *CellInventory) AddToInventory(itemType ItemType, value int16) int16 {
	c.inventoryMx.Lock()
	defer c.inventoryMx.Unlock()
	c.inventory[itemType] += value
	c.inventory[itemType] = max(c.minForItemType(itemType), c.inventory[itemType])
	c.inventory[itemType] = min(c.maxForItemType(itemType), c.inventory[itemType])
	return c.inventory[itemType]
}

func (c *CellInventory) minForItemType(itemType ItemType) int16 {
	return 0
}

func (c *CellInventory) maxForItemType(itemType ItemType) int16 {
	switch itemType {
	case ItemTypeEnergy:
		return MaxEnergy
	case ItemTypeWater:
		return WaterMaxAmount
	case ItemTypeOrganic:
		return MaxEnergy
	}
	panic(itemType)
}

type Cell struct {
	CellInventory
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
		c.getWater(w)
	case CellTypeConnector:
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
	w.inventoryMx.Lock()
	c.AddToInventory(ItemTypeEnergy, w.inventory[pos][ItemTypeEnergy])
	w.inventoryMx.Unlock()
}

func (c *Cell) tryToCreateSeed(w *World) {
	if c.flowerTimer > 0 {
		c.flowerTimer -= 1
		return
	}
	if c.inventory[ItemTypeEnergy] <= SeedSpawnEnergy+FlowerSpawnEnergy {
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

	newSeed := NewCell(futureGenome.id, CellTypeSeed, Inventory{
		ItemTypeEnergy: SeedSpawnEnergy, ItemTypeWater: c.inventory[ItemTypeWater] - WaterTransferAmount,
	}, uuid.NewString())
	c.inventory[ItemTypeEnergy] -= SeedSpawnEnergy
	newSeed.direction = direction
	c.inventory[ItemTypeWater] = WaterTransferAmount
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
	c.inventory[ItemTypeEnergy] = min(MaxEnergy, max(0, c.inventory[ItemTypeEnergy]-tax))
}

func (c *Cell) Die(w *World) {
	pos := w.cellPositions[c]
	w.inventory[pos][ItemTypeOrganic] += transformationEnergy(c.cellType) * rotMultiplier
	w.inventory[pos][ItemTypeWater] += c.GetFromInventory(ItemTypeWater)
	delete(w.cellPositions, c)
}

func (c *Cell) TooOld() bool {
	if c.cellType == CellTypeSeed {
		return c.age > MaxAge*3
	}
	return c.age > MaxAge
}

func (c *Cell) CheckEnergy(e int16) bool {
	return c.inventory[ItemTypeEnergy] > e
}

func (c *Cell) getOrganicEnergy(w *World) {
	got := w.DrainSquare(w.cellPositions[c], ItemTypeOrganic, OrganicDrainByCell)
	c.inventory[ItemTypeEnergy] += got
}

func (c *Cell) SpendWater(w *World) {
	waterTax := int16(2)
	if c.cellType == CellTypeFlower {
		waterTax += 1
	}
	if c.cellType == CellTypeSeed || c.cellType == CellTypeTrunk {
		waterTax = int16(0)
	}

	c.inventory[ItemTypeWater] = max(0, c.inventory[ItemTypeWater]-waterTax)
}

func (c *Cell) getWater(w *World) {
	if c.GetFromInventory(ItemTypeWater) >= WaterMaxAmount-WaterExtractionValue*4 {
		return
	}
	got := w.DrainSquare(w.cellPositions[c], ItemTypeWater, WaterExtractionValue)
	c.inventory[ItemTypeWater] += got
}

func NewCell(genomeID string, ct CellType, inventory Inventory, organismID string) *Cell {
	c := &Cell{cellType: ct, genomeID: genomeID, organismID: organismID}
	c.inventory = make(Inventory)
	for it := range inventory {
		c.inventory[it] = inventory[it]
	}
	return c
}
