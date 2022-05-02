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
		// Loses stem cell status after first mitosis, which is free.
		c := cell.Mitosis()
		c.Start(ctx)
	}
	return true
}

func ShouldMitosisAndRepair(ctx context.Context, cell *EukaryoticCell) bool {
	// Not all cells can repair, but for the sake of this simulation, they can.
	if cell.damage > 0 {
		resource := cell.parent.materialPool.GetResource()
		if resource.vitamins >= cell.damage {
			cell.damage = 0
			resource.vitamins -= cell.damage
		} else {
			cell.damage -= resource.vitamins
			resource.vitamins = 0
		}
		cell.parent.materialPool.PutResource(resource)

	}
	// Also not all cells can reproduce, but for sake of simulation, they can.
	// Three conditions for mitosis, to prevent runaway growth:
	// - No damage on the cell
	// - Enough growth ligand (signal)
	// - Enough vitamin resources
	if cell.damage == 0 {
		ligand := cell.parent.materialPool.GetLigand()
		if ligand.growth >= LIGAND_GROWTH_THRESHOLD {
			resource := cell.parent.materialPool.GetResource()
			if resource.vitamins >= VITAMIN_COST_MITOSIS {
				ligand.growth -= LIGAND_GROWTH_THRESHOLD
				resource.vitamins -= VITAMIN_COST_MITOSIS
				c := cell.Mitosis()
				c.Start(ctx)
			}
			cell.parent.materialPool.PutResource(resource)
		}
		cell.parent.materialPool.PutLigand(ligand)
	}
	return true
}

func ShouldApoptosis(ctx context.Context, cell *EukaryoticCell) bool {
	waste := cell.parent.materialPool.GetWaste()
	if waste.toxins >= TOXINS_THRESHOLD {
		cell.damage++
	}
	if waste.co2 >= CO2_THRESHOLD {
		cell.damage++
	}
	cell.parent.materialPool.PutWaste(waste)
	if cell.killSignal || cell.damage > MAX_DAMAGE {
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
		if waste.co2 > 6 {
			waste.co2 -= 6
		} else {
			waste.co2 = 0
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
	request := cell.parent.RequestWork(Work{
		workType: think,
	})
	if request.status == 200 {
		// Successfully found a food unit.
		cell.parent.RequestWork(Work{
			workType: digest,
		})
	}

	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: cover,
	})
	return true
}

func BrainStimulateMuscles(ctx context.Context, cell *EukaryoticCell) bool {
	// Check resources in the brain. If not enough,
	// stimulate muscle movements.
	resource := cell.parent.materialPool.GetResource()
	if resource.vitamins <= BRAIN_VITAMIN_THRESHOLD {
		// If vitamin levels are low, move more.
		cell.parent.RequestWork(Work{
			workType: move,
		})
	}
	cell.parent.materialPool.PutResource(resource)
	return true
}

func BrainRequestPump(ctx context.Context, cell *EukaryoticCell) bool {
	cell.parent.RequestWork(Work{
		workType: pump,
	})
	return true
}

func Digest(ctx context.Context, cell *EukaryoticCell) bool {
	// TODO: Make this process linked to gut bacteria.
	localResource := cell.parent.materialPool.GetLocalResource()
	if localResource.glucose > 0 {
		// Convert glucose to vitamins.
		resource := cell.parent.materialPool.GetResource()
		resource.vitamins += localResource.glucose
		cell.parent.materialPool.PutResource(resource)
	}
	cell.parent.materialPool.PutLocalResource(localResource)
	return true
}

func GenerateRandomProteinPermutation(c *EukaryoticCell) (proteins []Protein) {
	chooseN := len(proteins) / 3
	permutations := rand.Perm(chooseN)
	var allProteins []Protein
	for protein := range c.dna.selfProteins {
		allProteins = append(allProteins, protein)
	}
	for i := 0; i < chooseN; i++ {
		proteins = append(proteins, allProteins[permutations[i]])
	}
	return
}

func MakeStateDiagramByCell(c *EukaryoticCell) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   StemCellToSpecializedCell,
				proteins: GenerateRandomProteinPermutation(c),
			},
			next: &StateNode{
				next: nil,
				function: &ProteinFunction{
					action:   DoWork,
					proteins: GenerateRandomProteinPermutation(c),
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
				action:   Expirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	case Neuron:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   BrainRequestPump,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   BrainStimulateMuscles,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	case Enterocyte:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   Digest,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	case Pneumocyte:
		fallthrough
	case Keratinocyte:
		// Do nothing but work.
	case Myocyte:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   MuscleFindFood,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   MuscleSeekSkinProtection,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		fallthrough
	case Cardiomyocyte:
		fallthrough
	default:
		currNode.next = &StateNode{
			next: nil,
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: nil,
		function: &ProteinFunction{
			action:   ShouldMitosisAndRepair,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	currNode = currNode.next
	currNode.next = &StateNode{
		next: s.root.next, // Do Work
		function: &ProteinFunction{
			action:   ShouldApoptosis,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	return s
}
