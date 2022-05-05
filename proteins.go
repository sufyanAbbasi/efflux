package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type StateDiagram struct {
	sync.RWMutex
	root    *StateNode
	current *StateNode
}

func (s *StateDiagram) Run(ctx context.Context, cell CellActor) {
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
					s.Lock()
					s.current = s.current.next
					s.Unlock()
				} else {
					cancel()
				}
			}
		}
	}
}

func (s *StateDiagram) Graft(mutation *StateDiagram) {
	s.Lock()
	s.root = mutation.root
	s.current = mutation.root
	s.Unlock()
}

type CellActor interface {
	Worker
	AntigenPresenting
	CellType() CellType
	Start(context.Context)
	Parent() *Node
	Function() *StateDiagram
	HasTelomerase() bool
	ShouldMitosis() bool
	Mitosis() CellActor
	Damage() int
	Repair(int)
	IncurDamage(int)
	Apoptosis()
	IsAerobic() bool
	IsOxygenated() bool
	Oxygenate(bool)
}

type StateNode struct {
	next     *StateNode
	function *ProteinFunction
}

// Return false if terminal
type CellAction func(ctx context.Context, cell CellActor) bool

type ProteinFunction struct {
	proteins []Protein
	action   CellAction
}

func (p *ProteinFunction) Run(ctx context.Context, cell CellActor) bool {
	return p.action(ctx, cell)
}

// General Actions

func checkParentOrDie(ctx context.Context, cell CellActor) bool {
	if cell.Parent() == nil {
		fmt.Printf("Force killed: %v\n", cell)
		return Apoptosis(ctx, cell)
	}
	return true
}

func DoWork(ctx context.Context, cell CellActor) bool {
	if !cell.IsAerobic() || cell.IsOxygenated() {
		cell.Parent().MakeAvailable(cell)
		cell.Oxygenate(false)
	}
	return true
}

func StemCellToSpecializedCell(ctx context.Context, cell CellActor) bool {
	if cell.HasTelomerase() {
		// Loses stem cell status after first mitosis, which is free.
		c := cell.Mitosis()
		c.Start(ctx)
	}
	return true
}

func ShouldMitosisAndRepair(ctx context.Context, cell CellActor) bool {
	// Not all cells can repair, but for the sake of this simulation, they can.
	resource := cell.Parent().materialPool.GetResource()
	defer cell.Parent().materialPool.PutResource(resource)
	if cell.Damage() > 0 {
		repair := resource.vitamins
		if resource.vitamins >= cell.Damage() {
			resource.vitamins -= cell.Damage()
		} else {
			resource.vitamins = 0
		}
		cell.Repair(repair)
	}
	// Also not all cells can reproduce, but for sake of simulation, they can.
	// Three conditions for mitosis, to prevent runaway growth:
	// - No damage on the cell
	// - Enough growth ligand (signal)
	// - Enough vitamin resources
	if cell.Damage() <= DAMAGE_MITOSIS_THRESHOLD && cell.ShouldMitosis() {
		if resource.vitamins >= VITAMIN_COST_MITOSIS {
			resource.vitamins -= VITAMIN_COST_MITOSIS
			c := cell.Mitosis()
			c.Start(ctx)
		}
	}
	return true
}

func ShouldApoptosis(ctx context.Context, cell CellActor) bool {
	waste := cell.Parent().materialPool.GetWaste()
	defer cell.Parent().materialPool.PutWaste(waste)
	if waste.toxins >= TOXINS_THRESHOLD {
		cell.IncurDamage(1)
	}
	if waste.co2 >= CO2_THRESHOLD {
		cell.IncurDamage(1)
	}
	if cell.Damage() > MAX_DAMAGE {
		Apoptosis(ctx, cell)
		return false
	}
	return true
}

func Apoptosis(ctx context.Context, cell CellActor) bool {
	fmt.Println(cell, " Died!")
	cell.Apoptosis()
	return false
}

func Respirate(ctx context.Context, cell CellActor) bool {
	// Receive a unit of 02 for a unit of CO2
	request := cell.Parent().RequestWork(Work{
		workType: inhale,
	})
	if request.status == 200 {
		resource := cell.Parent().materialPool.GetResource()
		defer cell.Parent().materialPool.PutResource(resource)
		resource.o2 += 6
		waste := cell.Parent().materialPool.GetWaste()
		defer cell.Parent().materialPool.PutWaste(waste)
		if waste.co2 > 6 {
			waste.co2 -= 6
		} else {
			waste.co2 = 0
		}
		cell.Oxygenate(true)
	}
	return true
}

func Expirate(ctx context.Context, cell CellActor) bool {
	// Exchange a unit of C02 for a unit of O2
	request := cell.Parent().RequestWork(Work{
		workType: exhale,
	})
	if request.status == 200 {
		waste := cell.Parent().materialPool.GetWaste()
		defer cell.Parent().materialPool.PutWaste(waste)
		if waste.co2 <= 6 {
			waste.co2 = 0
		} else {
			waste.co2 -= 6
		}

		resource := cell.Parent().materialPool.GetResource()
		defer cell.Parent().materialPool.PutResource(resource)
		resource.o2 += 6
		cell.Oxygenate(true)
	}
	return true
}

func MuscleFindFood(ctx context.Context, cell CellActor) bool {
	request := cell.Parent().RequestWork(Work{
		workType: think,
	})
	if request.status == 200 {
		// Successfully found a food unit.
		cell.Parent().RequestWork(Work{
			workType: digest,
		})
	}

	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell CellActor) bool {
	cell.Parent().RequestWork(Work{
		workType: cover,
	})
	return true
}

func BrainStimulateMuscles(ctx context.Context, cell CellActor) bool {
	// Check resources in the brain. If not enough,
	// stimulate muscle movements.
	resource := cell.Parent().materialPool.GetResource()
	defer cell.Parent().materialPool.PutResource(resource)
	if resource.vitamins <= BRAIN_VITAMIN_THRESHOLD {
		// If vitamin levels are low, move more.
		cell.Parent().RequestWork(Work{
			workType: move,
		})
	}
	return true
}

func BrainRequestPump(ctx context.Context, cell CellActor) bool {
	cell.Parent().RequestWork(Work{
		workType: pump,
	})
	return true
}

func Flatulate(ctx context.Context, cell CellActor) bool {
	// Manage CO2 levels by leaking it.
	waste := cell.Parent().materialPool.GetWaste()
	defer cell.Parent().materialPool.PutWaste(waste)
	waste.co2 = 0
	return true
}

func GenerateRandomProteinPermutation(c CellActor) (proteins []Protein) {
	chooseN := len(proteins) / 3
	permutations := rand.Perm(chooseN)
	var allProteins []Protein
	for protein := range c.DNA().selfProteins {
		allProteins = append(allProteins, protein)
	}
	for i := 0; i < chooseN; i++ {
		proteins = append(proteins, allProteins[permutations[i]])
	}
	return
}

func MakeStateDiagramByEukaryote(c CellActor) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   StemCellToSpecializedCell,
				proteins: GenerateRandomProteinPermutation(c),
			},
		},
	}
	currNode := s.root
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   DoWork,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	currNode = currNode.next
	switch c.CellType() {
	case RedBlood:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Expirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	case Neuron:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BrainRequestPump,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BrainStimulateMuscles,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	case Enterocyte:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Flatulate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		for i := 3; i > 0; i-- {
			currNode = currNode.next
			// Larger demand for oxygen to supply gut bacteria.
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   Respirate,
					proteins: GenerateRandomProteinPermutation(c),
				},
			}
		}
	case Pneumocyte:
		fallthrough
	case Keratinocyte:
		// Do nothing but work.
	case Myocyte:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   MuscleFindFood,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
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
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
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

// Bacteria Related CellActions

func Digest(ctx context.Context, cell CellActor) bool {
	if cell.IsAerobic() && cell.IsOxygenated() {
		resource := cell.Parent().materialPool.GetResource()
		defer cell.Parent().materialPool.PutResource(resource)
		if resource.glucose > 0 {
			// Convert glucose to vitamins.
			resource := cell.Parent().materialPool.GetResource()
			defer cell.Parent().materialPool.PutResource(resource)
			resource.vitamins += resource.glucose
		}
		cell.Oxygenate(false)
	}
	return true
}

func BacteriaShouldMitosis(ctx context.Context, cell CellActor) bool {
	// Bacteria will not be allowed to repair itself.
	// Three conditions for bacteria mitosis, may allow runaway growth:
	// - Enough internal energy (successful calls to Oxygenate)
	// - Enough vitamin or glucose resources (not picky)
	// - Enough time has passed
	if cell.ShouldMitosis() {
		resource := cell.Parent().materialPool.GetResource()
		defer cell.Parent().materialPool.PutResource(resource)
		if resource.glucose >= GLUCOSE_COST_MITOSIS {
			resource.glucose -= GLUCOSE_COST_MITOSIS
			c := cell.Mitosis()
			c.Start(ctx)
		} else if resource.vitamins >= VITAMIN_COST_MITOSIS {
			resource.vitamins -= VITAMIN_COST_MITOSIS
			c := cell.Mitosis()
			c.Start(ctx)
		}
	}
	return true
}

func BacteriaConsume(ctx context.Context, cell CellActor) bool {
	if cell.IsAerobic() {
		// Consume a unit of 02 for a unit of CO2
		resource := cell.Parent().materialPool.GetResource()
		defer cell.Parent().materialPool.PutResource(resource)
		if resource.o2 > 6 {
			resource.o2 -= 6
			cell.Oxygenate(true)
			waste := cell.Parent().materialPool.GetWaste()
			defer cell.Parent().materialPool.PutWaste(waste)
			if waste.co2 > 6 {
				waste.co2 -= 6
			} else {
				waste.co2 = 0
			}
		}
	} else {
		waste := cell.Parent().materialPool.GetWaste()
		defer cell.Parent().materialPool.PutWaste(waste)
		if waste.co2 > 6 {
			waste.co2 -= 6
			waste.toxins += 1
		}
		cell.Oxygenate(true)
	}
	return true
}

func BacteriaShouldApoptosis(ctx context.Context, cell CellActor) bool {
	waste := cell.Parent().materialPool.GetWaste()
	defer cell.Parent().materialPool.PutWaste(waste)
	if waste.toxins >= TOXINS_THRESHOLD {
		cell.IncurDamage(1)
	}
	if waste.co2 >= CO2_THRESHOLD {
		cell.IncurDamage(1)
	}
	if cell.Damage() > MAX_DAMAGE {
		Apoptosis(ctx, cell)
		return false
	}
	return true
}

func MakeStateDiagramByProkaryote(c CellActor) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   BacteriaShouldMitosis,
				proteins: GenerateRandomProteinPermutation(c),
			},
		},
	}
	currNode := s.root
	switch c.CellType() {
	case Bacteroidota:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Digest,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   BacteriaConsume,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	currNode = currNode.next
	currNode.next = &StateNode{
		next: s.root, // Back to beginning.
		function: &ProteinFunction{
			action:   BacteriaShouldApoptosis,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	return s
}

// Virus StateDiagrams

func MakeVirusProtein(ctx context.Context, cell CellActor) bool {
	// TODO: Implement this.
	return true
}

func ProduceInterferon(ctx context.Context, cell CellActor) bool {
	// TODO: Implement this.
	return true
}

func ProduceVirus(c CellActor) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   MakeVirusProtein,
				proteins: GenerateRandomProteinPermutation(c),
			},
		},
	}
	currNode := s.root
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   ProduceInterferon,
			proteins: GenerateRandomProteinPermutation(c),
		},
		next: s.root,
	}
	return s
}
