package internal

const (
	WorldSize = 100

	MaxEnergy          = int16(1024)
	EnergyTax          = int16(2)
	SeedEnergy         = int16(50)
	TrunkSpawnEnergy   = int16(10)
	FlowerSpawnEnergy  = int16(10)
	LeafSpawnEnergy    = int16(8)
	RootSpawnEnergy    = int16(8)
	SeedSpawnEnergy    = int16(15)
	SproutSpawnEnergy  = int16(5)
	OrganicDrainByCell = int16(2)
	MaxAge             = 500
)

type Direction uint8

const (
	DirectionWest Direction = iota
	DirectionNorth
	DirectionEast
	DirectionSouth
	DirectionMax
)

const MutationChance float32 = 0.1

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
