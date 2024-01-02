package internal

const (
	WorldSize = 100

	MaxEnergy          = int16(1024)
	EnergyTax          = int16(2)
	TrunkSpawnEnergy   = int16(10)
	FlowerSpawnEnergy  = int16(50)
	LeafSpawnEnergy    = int16(8)
	RootSpawnEnergy    = int16(8)
	SeedSpawnEnergy    = int16(15)
	SproutSpawnEnergy  = int16(5)
	OrganicDrainByCell = int16(2)
	rotMultiplier      = 4
	MaxAge             = 1000

	StartingOrganicLevel  = 100
	MaxSunLevel           = 20
	MaxSeedFlyingDistance = 20
	EnergyTransferAmount  = 10
)

type Direction uint8

const (
	DirectionWest Direction = iota
	DirectionNorth
	DirectionEast
	DirectionSouth
	DirectionMax
)

const MutationChance float32 = 0.0001

type CellType uint8

const (
	CellTypeLeaf CellType = iota
	CellTypeTrunk
	CellTypeFlower
	CellTypeSeed
	CellTypeSprout
	CellTypeRoot
	MaxCellType
)

type ConditionType uint8

const (
	CompareNextTwoGenes ConditionType = iota
	CompareEnergyLevel
	CompareCellType
	CompareNeighboursCount
	MaxConditionType
)

type GeneCommand uint8

const (
	GenePass GeneCommand = iota
	GeneIf
	GeneGoTo
	GeneTurnTo
	GeneMove
	GeneRotate
	MaxGeneCommand
)

type Relation uint8

const (
	RelationSameOrganism Relation = iota
	RelationSameGenome
	RelationAnotherOrganism
	RelationAnotherGenome
	RelationAny
	MaxRelationType
)

func relationMatch(cell, another *Cell, r Relation) bool {
	switch r {
	case RelationSameOrganism:
		return cell.organismID == another.organismID
	case RelationSameGenome:
		return cell.genomeID == another.genomeID
	case RelationAnotherOrganism:
		return cell.organismID != another.organismID
	case RelationAnotherGenome:
		return cell.genomeID != another.genomeID
	case RelationAny:
		return true
	}
	panic("invalid relation")
}
