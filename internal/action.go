package internal

import "github.com/google/uuid"

type Action interface {
	Apply(cell *Cell, world *World)
}

type ActionDoNothing struct {
	nextGenomePosition uint8
}

func NewActionDoNothing(nextGenomePosition uint8) *ActionDoNothing {
	return &ActionDoNothing{nextGenomePosition: nextGenomePosition}
}

func (a *ActionDoNothing) Apply(cell *Cell, world *World) {
	cell.genomePosition = a.nextGenomePosition
}

func transformationEnergy(ct CellType) int16 {
	switch ct {
	case CellTypeTrunk:
		return TrunkSpawnEnergy
	case CellTypeFlower:
		return FlowerSpawnEnergy
	case CellTypeLeaf:
		return LeafSpawnEnergy
	case CellTypeRoot:
		return RootSpawnEnergy
	case CellTypeSeed:
		return SeedSpawnEnergy
	case CellTypeSprout:
		return SproutSpawnEnergy
	}
	panic(ct)
}

type ActionChangeCellType struct {
	target CellType

	sproutsAmount      uint8
	nextGenomePosition uint8
}

func NewActionChangeCellType(target CellType, nextGenomePosition, sproutsAmount uint8) *ActionChangeCellType {
	return &ActionChangeCellType{target: target, nextGenomePosition: nextGenomePosition, sproutsAmount: sproutsAmount}
}

func (a *ActionChangeCellType) Apply(cell *Cell, world *World) {
	cell.genomePosition = a.nextGenomePosition
	if cell.cellType == a.target || (a.target == CellTypeSeed && cell.cellType != CellTypeFlower) {
		return
	}
	if cell.cellType != CellTypeSeed && cell.cellType != CellTypeSprout {
		return
	}
	energyRequired := transformationEnergy(a.target)
	if a.target == CellTypeTrunk {
		energyRequired += int16(a.sproutsAmount) * SproutSpawnEnergy
	}
	if !cell.CheckEnergy(energyRequired) {
		return
	}
	cell.cellType = a.target
	cell.genomePosition = a.nextGenomePosition

	if a.target == CellTypeSeed {
		// flower turns into seed!
		cell.organismID = uuid.NewString()
		childGenome := NewGenome(world.GetGenome(cell.genomeID))
		cell.genomeID = childGenome.id
		cell.genomePosition = 0
		world.RegisterNewCell(cell, world.GetPosition(cell), childGenome)
		return
	}

	if a.target == CellTypeTrunk {
		ns := world.GetPosition(cell).Neighbours()
		spawned := uint8(0)
		for ix := range ns {
			if spawned == a.sproutsAmount {
				break
			}
			spawned += 1
			newSprout := NewCell(cell.genomeID, CellTypeSprout, TrunkSpawnEnergy, cell.organismID)
			newSprout.direction = Direction(ix)
			newSprout.genomePosition = cell.genomePosition
			world.RegisterNewCell(newSprout, ns[ix], world.GetGenome(cell.genomeID))
		}
	}
}

type ActionMove struct {
	nextGenomePosition uint8
}

func (a *ActionMove) Apply(cell *Cell, world *World) {
	if cell.cellType == CellTypeSprout {
		return
	}
	world.RegisterMove(cell)
	cell.genomePosition = a.nextGenomePosition
}

func NewActionMove(nextGenomePosition uint8) *ActionMove {
	return &ActionMove{nextGenomePosition: nextGenomePosition}
}

type ActionRotate struct {
	left               bool
	nextGenomePosition uint8
}

func NewActionRotate(left bool, nextGenomePosition uint8) *ActionRotate {
	return &ActionRotate{nextGenomePosition: nextGenomePosition, left: left}
}

func (a *ActionRotate) Apply(cell *Cell, world *World) {
	if a.left {
		cell.direction -= 1
	} else {
		cell.direction += 1
	}
	cell.direction = cell.direction % DirectionMax
	cell.genomePosition = a.nextGenomePosition
}
