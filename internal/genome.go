package internal

import (
	"github.com/google/uuid"
	"math/rand"
	"sync"
)

type Genome struct {
	genome   []uint8
	id       string
	parentID string
}

func (g *Genome) extractCommand(gene uint8) GeneCommand {
	return GeneCommand(gene) % MaxGeneCommand
}

func (g *Genome) ExecutePosition(position uint8, world *World) Action {
	gene := g.genome[position]
	geneCommand := g.extractCommand(gene)
	switch geneCommand {
	case GenePass:
		return NewActionDoNothing(position + 1)
	case GeneIf:
		return g.executeIf(position)
	case GeneGoTo:
		return g.executeGoTo(position)
	case GeneTurnTo:
		return g.executeTurnTo(position)
	case GeneMove:
		return g.executeMove(position)
	case GeneRotate:
		return g.executeRotate(position)
	}
	return NewActionDoNothing(position + 1)
}

func (g *Genome) executeIf(position uint8) Action {
	conditionType := g.extractCondition(g.genome[position+1])
	switch conditionType {
	case CompareNextTwoGenes:
		first, second, lessEqual := g.genome[position+2], g.genome[position+3], g.genome[position+4]
		lessEqual = lessEqual % 2
		if lessEqual == 0 {
			// means we return the first gene's position if it is greater
			if first > second {
				return NewActionDoNothing(position + 2)
			} else {
				return NewActionDoNothing(position + 3)
			}
		} else {
			// means we return the first gene's position if it is le
			if first <= second {
				return NewActionDoNothing(position + 2)
			} else {
				return NewActionDoNothing(position + 3)
			}
		}
	case CompareEnergyLevel:
		value := (int16(g.GetGene(position+1)) / 255) * MaxEnergy
		less := g.GetGene(position+2)%2 == 0
		positionIfTrue := position + 3
		// to allow for goto statement at least
		positionIfFalse := position + 5
		comp := func(c *Cell, w *World) bool {
			return less && c.energy < value || !less && c.energy >= value
		}
		return NewActionCompareForCell(comp, positionIfTrue, positionIfFalse)
	case CompareCellType:
		value := CellType(g.GetGene(position+1) % uint8(MaxCellType))
		eq := g.GetGene(position+2)%2 == 0
		positionIfTrue := position + 3
		positionIfFalse := position + 5
		comp := func(c *Cell, w *World) bool {
			return eq && c.cellType == value || !eq && c.cellType != value
		}
		return NewActionCompareForCell(comp, positionIfTrue, positionIfFalse)
	case CompareNeighboursCount:
		maxNeighbours := uint8(4) + 1
		value := int(g.GetGene(position+1) % maxNeighbours)
		less := g.GetGene(position+2)%2 == 0
		relation := Relation(g.GetGene(position+3) % uint8(MaxRelationType))
		positionIfTrue := position + 4
		positionIfFalse := position + 6
		comp := func(c *Cell, w *World) bool {
			ns := w.GetPosition(c).Neighbours()
			nsCount := 0
			for i := range ns {
				anotherCell := w.GetCellByPosition(ns[i])
				if anotherCell == nil {
					continue
				}
				if w.Occupied(ns[i]) && relationMatch(c, anotherCell, relation) {
					nsCount += 1
				}
			}
			return less && nsCount < value || !less && nsCount >= value
		}
		return NewActionCompareForCell(comp, positionIfTrue, positionIfFalse)
	}

	return NewActionDoNothing(position + 1)
}

func (g *Genome) executeGoTo(position uint8) Action {
	whereToGo := g.GetGene(position + 1)
	return NewActionDoNothing(whereToGo)
}

func (g *Genome) extractCondition(u uint8) ConditionType {
	return ConditionType(u % uint8(MaxConditionType))
}

func (g *Genome) executeTurnTo(position uint8) Action {
	ct := CellType(g.GetGene(position+1) % uint8(MaxCellType))
	newPosition := position + 1
	if ct == CellTypeSeed {
		ct = CellType(rand.Uint32() % uint32(MaxCellType))
		if ct == CellTypeSeed {
			ct = (ct + 1) % MaxCellType
		}
	}
	sproutsAmount := g.GetGene(position+1)%4 + 1
	return NewActionChangeCellType(ct, newPosition, sproutsAmount, int(g.GetGene(position+2)%MaxSeedFlyingDistance))
}

func (g *Genome) extractTurningTarget(u uint8) CellType {
	return CellType(u % uint8(MaxCellType))
}

func (g *Genome) GetID() string {
	return g.id
}

func (g *Genome) executeMove(position uint8) Action {
	return NewActionMove(position + 1)
}

func (g *Genome) executeRotate(position uint8) Action {
	direction := g.genome[position+1] % 2
	left := false
	if direction == 1 {
		left = true
	}
	return NewActionRotate(left, position+1)
}

func (g *Genome) GetGene(position uint8) uint8 {
	return g.genome[position]
}

func (g *Genome) Copy(w *World) *Genome {
	childGenome := NewGenome(g)
	if childGenome.id != g.id {
		w.GenomeStorage.AddGenome(childGenome)
	}
	return childGenome
}

func NewGenome(parentGenome *Genome) *Genome {
	g := Genome{id: uuid.NewString(), genome: make([]uint8, 256), parentID: ""}
	if parentGenome == nil {
		for i := range g.genome {
			g.genome[i] = uint8(rand.Uint32())
		}
		return &g
	}

	changes := false
	g.genome = parentGenome.genome
	for i := range g.genome {
		if rand.Float32() < MutationChance {
			g.genome[i] = uint8(rand.Uint32())
			changes = true
		}
	}
	if changes {
		g.parentID = parentGenome.parentID
	} else {
		return parentGenome
	}
	return &g
}

type GenomeStorage struct {
	genomes    map[string]*Genome
	newGenomes map[string]*Genome
	genomeMx   sync.Mutex
}

func NewGenomeStorage() GenomeStorage {
	return GenomeStorage{genomes: make(map[string]*Genome), newGenomes: make(map[string]*Genome)}
}

func (s *GenomeStorage) AddGenome(g *Genome) {
	s.genomeMx.Lock()
	s.genomes[g.id] = g
	s.genomeMx.Unlock()
}

func (s *GenomeStorage) GetGenome(id string) *Genome {
	s.genomeMx.Lock()
	defer s.genomeMx.Unlock()
	return s.genomes[id]
}
