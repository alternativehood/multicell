package internal

import (
	"fmt"
	"sync"
)

type Organism struct {
	cells            []*Cell
	transferredCells []*Cell
}

func (o *Organism) RegisterCell(cell *Cell) {
	o.cells = append(o.cells, cell)
}

func (o *Organism) HandleEnergyFlow(world *World, wg *sync.WaitGroup) {
	defer wg.Done()
	if len(o.cells) <= 1 {
		return
	}
	var wgC sync.WaitGroup
	totalEnergy := int16(0)
	for i := range o.cells {
		totalEnergy += o.cells[i].energy
	}
	var mx sync.Mutex
	wgC.Add(len(o.cells))
	type energyChange struct {
		c      *Cell
		change int16
	}
	energyChanges := make([]energyChange, 0)
	for ix := range o.cells {
		cell := o.cells[ix]
		go func() {
			defer wgC.Done()
			if cell.energy < 2*EnergyTransferAmount {
				return
			}
			pos := world.GetPosition(cell)
			ns := pos.Neighbours()
			var poorestFriend *Cell
			for i := range ns {
				n := world.GetCellByPosition(ns[i])
				if n != nil && n.organismID == cell.organismID {
					if poorestFriend == nil {
						poorestFriend = n
						continue
					}
					if poorestFriend.energy > n.energy {
						poorestFriend = n
					}
				}
			}
			if poorestFriend == nil {
				return
			}
			if poorestFriend.energy >= cell.energy {
				return
			}
			mx.Lock()
			energyChanges = append(
				energyChanges, energyChange{c: cell, change: -EnergyTransferAmount},
				energyChange{c: poorestFriend, change: EnergyTransferAmount},
			)
			mx.Unlock()
		}()
	}
	wgC.Wait()

	for ix := range energyChanges {
		energyChanges[ix].c.energy += energyChanges[ix].change
	}

	newTotalEnergy := int16(0)
	for i := range o.cells {
		newTotalEnergy += o.cells[i].energy
	}
	if newTotalEnergy != totalEnergy {
		panic(fmt.Errorf("energy changed after spreading"))
	}
}
