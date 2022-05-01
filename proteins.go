package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type StateDiagram struct {
	root    *StateNode
	current *StateNode
}

func (s *StateDiagram) Run(ctx context.Context, cell *EukaryoticCell) {
	// Add a random delay to offset cells.
	time.Sleep(time.Duration(rand.Float32()*100) * time.Millisecond)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ticker := time.NewTicker(CELL_CLOCK_RATE)
	if s.root != nil {
		s.current = s.root
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
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
	if cell.hasTelomerase {
		// Loses stem cell status after first mitosis.
		c := cell.Mitosis()
		fmt.Println(c)
		c.Start(ctx)
	}
	return true
}

func ShouldMitosis(ctx context.Context, cell *EukaryoticCell) bool {
	// TODO: Check some conditions for cell reproduction.
	if false {
		c := cell.Mitosis()
		c.Start(ctx)
	}
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
	fmt.Println(cell, " Died!")
	cell.Apoptosis()
	return false
}

func Respirate(ctx context.Context, cell *EukaryoticCell) bool {
	// Receive a unit of 02 for a unit of CO2
	request := cell.parent.RequestWork(Work{
		workType: inhale,
	})
	if request.status == 200 {
		resource := cell.parent.materialPool.GetResource()
		resource.o2 += 6
		cell.parent.materialPool.PutResource(resource)
		waste := cell.parent.materialPool.GetWaste()
		if waste.co2 <= 6 {
			waste.co2 = 0
		} else {
			waste.co2 -= 6
		}
		cell.parent.materialPool.PutWaste(waste)
	}
	return true
}

func Expirate(ctx context.Context, cell *EukaryoticCell) bool {
	// Exchange a unit of C02 for a unit of O2
	request := cell.parent.RequestWork(Work{
		workType: exhale,
	})
	if request.status == 200 {
		waste := cell.parent.materialPool.GetWaste()
		if waste.co2 <= 6 {
			waste.co2 = 0
		} else {
			waste.co2 -= 6
		}
		cell.parent.materialPool.PutWaste(waste)

		resource := cell.parent.materialPool.GetResource()
		resource.o2 += 6
		cell.parent.materialPool.PutResource(resource)
	}
	return true
}

func MuscleFindFood(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: think,
	})
	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: cover,
	})
	return true
}

func BrainControlMuscles(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: move,
	})
	return true
}

func BrainRequestPump(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: pump,
	})
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
				action: Expirate,
			},
		}
		currNode = currNode.next
	case Neuron:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: BrainRequestPump,
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: Respirate,
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: BrainControlMuscles,
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: Respirate,
			},
		}
		currNode = currNode.next
	case Keratinocyte:
		// Do nothing.
	case Myocyte:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: MuscleFindFood,
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: MuscleSeekSkinProtection,
			},
		}
		currNode = currNode.next
		fallthrough
	case Cardiomyocyte:
		fallthrough
	case Pneumocyte:
		fallthrough
	default:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action: Respirate,
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: nil,
		function: &ProteinFunction{
			action: ShouldMitosis,
		},
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
