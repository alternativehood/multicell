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
		newPosition = g.GetGene(position + 1)
	}
	sproutsAmount := uint8(0)
	if ct == CellTypeTrunk {
		sproutsAmount = g.GetGene(position+1)%4 + 1
	}
	return NewActionChangeCellType(ct, newPosition, sproutsAmount)
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
		if rand.Float32() > MutationChance {
			g.genome[i] = uint8(rand.Uint32())
			changes = true
		}
	}
	if changes {
		g.parentID = parentGenome.parentID
		g.id = uuid.NewString()
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
	return s.genomes[id]
}
