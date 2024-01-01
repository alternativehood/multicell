package internal

import (
	"fmt"
	"sync"
	"time"
)

type Position struct {
	X, Y int64
}

func NewPosition(x, y int64) Position {
	return Position{X: (WorldSize + x) % WorldSize, Y: (WorldSize + y) % WorldSize}
}

func (p Position) Equal(other Position) bool {
	return p.X == other.X && p.Y == other.Y
}

func (p Position) Neighbours() []Position {
	return []Position{
		p.MovedByDirection(DirectionWest),
		p.MovedByDirection(DirectionNorth),
		p.MovedByDirection(DirectionEast),
		p.MovedByDirection(DirectionSouth),
	}
}

func (p Position) MovedByDirection(direction Direction) Position {
	x, y := p.X, p.Y
	switch direction {
	case DirectionWest:
		x -= 1
	case DirectionEast:
		x += 1
	case DirectionNorth:
		y -= 1
	case DirectionSouth:
		y += 1
	}
	return NewPosition(x, y)
}

type LocationProperties struct {
	sunlight int16
	organic  int16
}

type WorldExport struct {
	cellTypes map[Position]CellType
	energy    map[Position]int16
	turn      int
}

func (e *WorldExport) CellTypes() map[Position]CellType {
	return e.cellTypes
}

func (e *WorldExport) Energy() map[Position]int16 {
	return e.energy
}

func (e *WorldExport) Turn() int {
	return e.turn
}

func NewWorldExport() WorldExport {
	return WorldExport{cellTypes: make(map[Position]CellType), energy: make(map[Position]int16)}
}

type World struct {
	GenomeStorage
	properties    map[Position]*LocationProperties
	cellPositions map[*Cell]Position
	size          int64

	moveAttempts []*Cell
	newCells     map[*Cell]Position

	newCellsMx sync.Mutex
	movesMx    sync.Mutex
	organisms  map[string][]string
	turn       int
}

func (w *World) GetCellByPosition(pos Position) *Cell {
	for ix := range w.cellPositions {
		if w.cellPositions[ix] == pos {
			return ix
		}
	}
	return nil
}

func (w *World) AddCell(pos Position, cell *Cell) {
	w.cellPositions[cell] = pos
}

func (w *World) Occupied(pos Position) bool {
	for ix := range w.cellPositions {
		if w.cellPositions[ix] == pos {
			return true
		}
	}
	return false
}

func (w *World) GetProperties(pos Position) *LocationProperties {
	return w.properties[pos]
}

func (w *World) RegisterMove(c *Cell) {
	w.movesMx.Lock()
	w.moveAttempts = append(w.moveAttempts, c)
	w.movesMx.Unlock()
}

func (w *World) CleanupTurn() {
	w.moveAttempts = make([]*Cell, 0)
	w.newCells = make(map[*Cell]Position)
	w.newGenomes = make(map[string]*Genome)
	w.turn += 1
}

func (w *World) GetPosition(c *Cell) Position {
	return w.cellPositions[c]
}

func (w *World) RegisterNewCell(cell *Cell, position Position, genome *Genome) {
	w.newCellsMx.Lock()
	w.newCells[cell] = position
	w.newGenomes[genome.id] = genome
	w.newCellsMx.Unlock()
}

func (w *World) ExecuteCellGenomes() {
	var wg sync.WaitGroup
	start := time.Now()
	wg.Add(len(w.cellPositions))
	for cell := range w.cellPositions {
		go cell.ExecuteGenome(w, &wg)
	}
	wg.Wait()
	print(fmt.Sprintf("Elapsed time thinking: %f\n", time.Now().Sub(start).Seconds()))
}

func (w *World) ExecuteTypeActions() {
	var wg sync.WaitGroup
	start := time.Now()
	wg.Add(len(w.cellPositions))
	for cell := range w.cellPositions {
		go cell.ExecuteTypeAction(w, &wg)
	}
	wg.Wait()
	print(fmt.Sprintf("Elapsed time type action: %f\n", time.Now().Sub(start).Seconds()))
}

func (w *World) MoveCells() {
	for ix := range w.moveAttempts {
		cell := w.moveAttempts[ix]
		currentPosition := w.cellPositions[cell]
		newPosition := currentPosition.MovedByDirection(cell.direction)
		if w.Occupied(newPosition) {
			continue
		}
		w.cellPositions[cell] = newPosition
	}
}

func (w *World) CreateNewCells() {
	for cell := range w.newCells {
		position := w.newCells[cell]
		if w.Occupied(position) {
			continue
		}
		w.GenomeStorage.AddGenome(w.newGenomes[cell.genomeID])
		w.cellPositions[cell] = position
	}
}

func (w *World) Export() WorldExport {
	result := NewWorldExport()
	for cell := range w.cellPositions {
		result.cellTypes[w.cellPositions[cell]] = cell.cellType
		result.energy[w.cellPositions[cell]] = cell.energy
	}
	result.turn = w.turn
	return result
}

func (w *World) RemoveCells() {
	for cell := range w.cellPositions {
		cell.age += 1
		if cell.energy <= 0 || cell.TooOld() {
			cell.Die(w)
		}
	}
}

func (w *World) DrainEnergy() {
	for cell := range w.cellPositions {
		cell.SpendEnergy(w)
	}
}

func (w *World) GetCells() map[*Cell]Position {
	return w.cellPositions
}

func (w *World) SpreadEnergy() {
	organisms := make(map[string]*Organism)
	for ix := range w.GetCells() {
		if _, found := organisms[ix.organismID]; !found {
			organisms[ix.organismID] = &Organism{}
		}
		organisms[ix.organismID].RegisterCell(ix)
	}
	for i := range organisms {
		organisms[i].HandleEnergyFlow()
	}
}

func (w *World) DrainOrganic(p Position) int16 {
	ns := p.Neighbours()
	allPositions := append(ns, p)
	totalGot := int16(0)
	for i := range allPositions {
		extraction := min(w.GetProperties(allPositions[i]).organic, OrganicDrainByCell)
		totalGot += extraction
		w.properties[allPositions[i]].organic -= extraction
	}
	return totalGot
}

func NewWorld(size int64) *World {
	w := World{
		size: size, properties: make(map[Position]*LocationProperties), cellPositions: make(map[*Cell]Position),
		moveAttempts: make([]*Cell, 0), newCells: make(map[*Cell]Position),
		GenomeStorage: NewGenomeStorage(),
	}
	for i := 0; i < WorldSize; i++ {
		for j := 0; j < WorldSize; j++ {
			pos := NewPosition(int64(i), int64(j))
			p := LocationProperties{sunlight: 10, organic: 5}
			w.properties[pos] = &p
		}
	}
	return &w
}
