package internal

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

type ActionCompareForCell struct {
	comp                            func(c *Cell, w *World) bool
	positionIfTrue, positionIfFalse uint8
}

func (a *ActionCompareForCell) Apply(cell *Cell, world *World) {
	if a.comp(cell, world) {
		cell.genomePosition = a.positionIfTrue
	} else {
		cell.genomePosition = a.positionIfFalse
	}
}

func NewActionCompareForCell(
	comp func(c *Cell, w *World) bool, positionIfTrue, positionIfFalse uint8,
) *ActionCompareForCell {
	return &ActionCompareForCell{
		comp: comp, positionIfTrue: positionIfTrue, positionIfFalse: positionIfFalse,
	}
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
	seedFlyingTimer    int
}

func NewActionChangeCellType(
	target CellType, nextGenomePosition, sproutsAmount uint8, seedFlyingTimer int,
) *ActionChangeCellType {
	return &ActionChangeCellType{
		target: target, nextGenomePosition: nextGenomePosition, sproutsAmount: sproutsAmount,
		seedFlyingTimer: seedFlyingTimer,
	}
}

func (a *ActionChangeCellType) Apply(cell *Cell, world *World) {
	cell.genomePosition = a.nextGenomePosition
	if cell.cellType == a.target || a.target == CellTypeSeed {
		return
	}
	if cell.cellType != CellTypeSeed && cell.cellType != CellTypeSprout {
		return
	}

	if cell.cellType == CellTypeSeed {
		// seed can only turn into trunk
		a.target = CellTypeTrunk
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

	if a.target == CellTypeFlower {
		cell.flowerTimer = int(FlowerSpawnEnergy)
		cell.seedFlyingTimer = a.seedFlyingTimer
	}

	if a.target == CellTypeTrunk {
		spawned := uint8(0)

		for i := 0; i < int(DirectionMax); i++ {
			if spawned == a.sproutsAmount {
				break
			}
			spawned += 1
			futureGenome := world.GetGenome(cell.genomeID).Copy(world)
			newSprout := NewCell(futureGenome.id, CellTypeSprout, TrunkSpawnEnergy, cell.organismID)
			newSprout.direction = Direction(i)
			newSprout.genomePosition = cell.genomePosition + uint8(i)
			world.RegisterNewCell(
				newSprout, world.GetPosition(cell).MovedByDirection(newSprout.direction), futureGenome,
			)
		}
	}
}

type ActionMove struct {
	nextGenomePosition uint8
}

func (a *ActionMove) Apply(cell *Cell, world *World) {
	cell.genomePosition = a.nextGenomePosition
	if cell.cellType == CellTypeSprout {
		return
	}
	world.RegisterMove(cell)
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
