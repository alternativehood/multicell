package internal

import (
	"sync"
)

type Organism struct {
	cells            []*Cell
	transferredCells []*Cell
}

func (o *Organism) RegisterCell(cell *Cell) {
	o.cells = append(o.cells, cell)
}

func (o *Organism) shouldTransferEnergy(world *World, from *Cell, to *Cell) bool {
	if from.cellType == CellTypeConnector {
		relation := world.GetGenome(from.genomeID).GetGene(from.genomePosition) % uint8(MaxRelationType)
		return relationMatch(from, to, Relation(relation))
	}

	if to.cellType == CellTypeConnector {
		relation := world.GetGenome(to.genomeID).GetGene(to.genomePosition) % uint8(MaxRelationType)
		return relationMatch(to, from, Relation(relation))
	}

	return relationMatch(from, to, RelationSameOrganism)
}

func (o *Organism) calculateDonationTarget(w *World, cell *Cell, ns []Position, itemType ItemType) *Cell {
	var poorestFriend *Cell
	for i := range ns {
		n := w.GetCellByPosition(ns[i])
		if n != nil {
			if !o.shouldTransferEnergy(w, cell, n) {
				continue
			}

			if poorestFriend == nil {
				poorestFriend = n
				continue
			}
			if poorestFriend.GetFromInventory(itemType) > n.GetFromInventory(itemType) {
				poorestFriend = n
			}
		}
	}
	if poorestFriend == nil {
		return nil
	}
	if poorestFriend.GetFromInventory(itemType) >= cell.GetFromInventory(itemType) {
		return nil
	}
	return poorestFriend
}

func (o *Organism) HandleResourcesFlow(world *World, wg *sync.WaitGroup) {
	defer wg.Done()
	if len(o.cells) <= 1 {
		return
	}
	var wgC sync.WaitGroup
	var changesMx sync.Mutex
	wgC.Add(len(o.cells))
	type change struct {
		c      *Cell
		change int16
	}
	changes := make(map[ItemType][]change)
	for i := ItemType(0); i < MaxItemType; i++ {
		changes[i] = make([]change, 0)
	}
	for ix := range o.cells {
		cell := o.cells[ix]
		go func() {
			defer wgC.Done()
			if cell.cellType == CellTypeFlower {
				// flowers only receive resources
				return
			}
			pos := world.GetPosition(cell)
			ns := pos.Neighbours()
			for it := ItemType(0); it < MaxItemType; it++ {
				if cell.GetFromInventory(it) < 2*itemSpreadStep(it) {
					continue
				}

				target := o.calculateDonationTarget(world, cell, ns, it)
				if target == nil {
					continue
				}
				changesMx.Lock()
				changes[it] = append(
					changes[it], change{c: cell, change: -itemSpreadStep(it)},
					change{c: target, change: itemSpreadStep(it)},
				)
				changesMx.Unlock()
			}
		}()
	}
	wgC.Wait()

	for it := range changes {
		for i := range changes[it] {
			changes[it][i].c.AddToInventory(it, changes[it][i].change)
		}
	}
}
