package main

import (
	"context"
	"fmt"
)

type StateDiagram struct {
	root    *StateNode
	current *StateNode
}

func (s *StateDiagram) Run(ctx context.Context, cell *EukaryoticCell) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if s.root != nil {
		s.current = s.root
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if s.current != nil {
					if checkParentOrDie(ctx, cell) && s.current.function != nil {
						if !s.current.function.Run(ctx, cell) {
							cancel()
						}
					} else {
						cancel()
					}
					s.current = s.current.next
				} else {
					cancel()
				}
			}
		}
	}
}

type StateNode struct {
	next     *StateNode
	function *ProteinFunction
}

// Return false if terminal
type CellAction func(ctx context.Context, cell *EukaryoticCell) bool

type ProteinFunction struct {
	proteins []Protein
	action   CellAction
}

func (p *ProteinFunction) Run(ctx context.Context, cell *EukaryoticCell) bool {
	return p.action(ctx, cell)
}

// General Actions

func checkParentOrDie(ctx context.Context, cell *EukaryoticCell) bool {
	if cell.parent == nil {
		fmt.Printf("Force killed: %v\n", cell)
		return Apoptosis(ctx, cell)
	}
	return true
}

func DoWork(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.MakeAvailable(cell)
	return true
}

func StemCellToSpecializedCell(ctx context.Context, cell *EukaryoticCell) bool {
	cell.Mitosis()
	// Loses stem cell status after first mitosis.
	cell.hasTelomerase = false
	return true
}

func ShouldApoptosis(ctx context.Context, cell *EukaryoticCell) bool {
	// TODO: Check some conditions for cell death.
	if false {
		Apoptosis(ctx, cell)
		return false
	}
	return true
}

func Apoptosis(ctx context.Context, cell *EukaryoticCell) bool {
	fmt.Println("Apoptosis")
	cell.Apoptosis()
	return false
}

func ConsumeOxygen(ctx context.Context, cell *EukaryoticCell) bool {
	// Receive a unit of 02 for a unit of CO2
	result := cell.parent.RequestWork(Work{
		workType: inhale,
	})
	if result.status == 200 {
		cell.Lock()
		cell.co2--
		cell.o2++
		cell.Unlock()
	} else {
		// Something bad's going on.
	}
	return true
}

func BloodExchangeGases(ctx context.Context, cell *EukaryoticCell) bool {
	if cell.co2 > 0 {
		// Exchange a unit of C02 for a unit of O2
		result := cell.parent.RequestWork(Work{
			workType: exhale,
		})
		if result.status == 200 {
			cell.Lock()
			cell.co2--
			cell.o2++
			cell.Unlock()
		} else {
			// No lung node available. Try exchanging with another blood.
			result := cell.parent.RequestWork(Work{
				workType: inhale,
			})
			if result.status == 200 {
				cell.Lock()
				cell.co2--
				cell.Unlock()
			} else {
				// We're in trouble. Try again later.
			}
		}
	}
	return true
}

func MakeStateDiagramByCell(c *EukaryoticCell) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action: StemCellToSpecializedCell,
			},
			next: &StateNode{
				next: nil,
				function: &ProteinFunction{
					action: DoWork,
				},
			},
		},
		current: nil,
	}
	currNode := s.root.next
	switch c.cellType {
	case RedBlood:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: BloodExchangeGases,
			},
		}
	default:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: ConsumeOxygen,
			},
		}
	}
	currNode = currNode.next
	currNode.next = &StateNode{
		next: s.root.next, // Do Work
		function: &ProteinFunction{
			action: ShouldApoptosis,
		},
	}
	return s
}
