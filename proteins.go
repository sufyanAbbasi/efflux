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
					if hasOrganOrDie(ctx, cell) && s.current.function != nil {
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
	Organ() *Node
	Function() *StateDiagram
	HasTelomerase() bool
	WillMitosis() bool
	Mitosis(ctx context.Context) CellActor
	CollectResources(context.Context) bool
	ProduceWaste()
	Damage() int
	Repair(int)
	IncurDamage(int)
	Apoptosis()
	IsAerobic() bool
	IsOxygenated() bool
	Oxygenate(bool)
	CanMove() bool
	Move(dx, dy, dz int)
	MoveTowardsCytokine(CytokineType) bool
	MoveAwayFromCytokine(CytokineType) bool
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

func hasOrganOrDie(ctx context.Context, cell CellActor) bool {
	if cell.Organ() == nil {
		fmt.Printf("Force killed: %v\n", cell)
		return Apoptosis(ctx, cell)
	}
	return true
}

func DoWork(ctx context.Context, cell CellActor) bool {
	cell.Organ().MakeAvailable(cell)
	return true
}

func MoveRandomly(ctx context.Context, cell CellActor) bool {
	cell.Move(1-rand.Intn(3), 1-rand.Intn(3), 0)
	return true
}

func MoveTowardsChemotaxisCytokine(ctx context.Context, cell CellActor) bool {
	cell.MoveTowardsCytokine(induce_chemotaxis)
	return true
}

func MoveTowardsCellDamangeCytokine(ctx context.Context, cell CellActor) bool {
	cell.MoveTowardsCytokine(cell_damage)
	return true
}

func MoveTowardsAntigenPresentCytokine(ctx context.Context, cell CellActor) bool {
	cell.MoveTowardsCytokine(antigen_present)
	return true
}

func StemCellToSpecializedCell(ctx context.Context, cell CellActor) bool {
	if cell.HasTelomerase() {
		// Loses stem cell status after first mitosis, which is free.
		cell.Mitosis(ctx)
	}
	return true
}

func WillMitosisAndRepair(ctx context.Context, cell CellActor) bool {
	// Not all cells can repair, but for the sake of this simulation, they can.
	resource := cell.Organ().materialPool.GetResource()
	defer cell.Organ().materialPool.PutResource(resource)
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
	// - Enough vitamin resources
	// - Enough growth ligand (signal)
	if cell.Damage() <= DAMAGE_MITOSIS_THRESHOLD &&
		resource.vitamins >= VITAMIN_COST_MITOSIS &&
		cell.WillMitosis() {
		resource.vitamins -= VITAMIN_COST_MITOSIS
		cell.Mitosis(ctx)
	}
	return true
}

func ShouldApoptosis(ctx context.Context, cell CellActor) bool {
	waste := cell.Organ().materialPool.GetWaste()
	defer cell.Organ().materialPool.PutWaste(waste)
	if waste.creatinine >= CREATININE_THRESHOLD {
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
	fmt.Println(cell, " Died in", cell.Organ())
	cell.Apoptosis()
	return false
}

func Respirate(ctx context.Context, cell CellActor) bool {
	// Receive a unit of 02 for a unit of CO2
	request := cell.Organ().RequestWork(Work{
		workType: exchange,
	})
	if request.status == 200 {
		resource := cell.Organ().materialPool.GetResource()
		defer cell.Organ().materialPool.PutResource(resource)
		resource.o2 += CELLULAR_TRANSPORT_O2
		resource.glucose += CELLULAR_TRANSPORT_GLUCOSE
		waste := cell.Organ().materialPool.GetWaste()
		defer cell.Organ().materialPool.PutWaste(waste)
		if waste.co2 > CELLULAR_TRANSPORT_CO2 {
			waste.co2 -= CELLULAR_TRANSPORT_CO2
		} else {
			waste.co2 = 0
		}
	}
	return true
}

func Expirate(ctx context.Context, cell CellActor) bool {
	// Exchange a unit of C02 for a unit of O2
	request := cell.Organ().RequestWork(Work{
		workType: exhale,
	})
	if request.status == 200 {
		waste := cell.Organ().materialPool.GetWaste()
		defer cell.Organ().materialPool.PutWaste(waste)
		if waste.co2 <= CELLULAR_TRANSPORT_CO2 {
			waste.co2 = 0
		} else {
			waste.co2 -= CELLULAR_TRANSPORT_CO2
		}

		resource := cell.Organ().materialPool.GetResource()
		defer cell.Organ().materialPool.PutResource(resource)
		resource.o2 += CELLULAR_TRANSPORT_O2
	}
	return true
}

func Filtrate(ctx context.Context, cell CellActor) bool {
	// Remove some amount of creatinine.
	request := cell.Organ().RequestWork(Work{
		workType: filter,
	})
	if request.status == 200 {
		waste := cell.Organ().materialPool.GetWaste()
		defer cell.Organ().materialPool.PutWaste(waste)
		if waste.creatinine <= CREATININE_FILTRATE {
			waste.creatinine = 0
		} else {
			waste.creatinine -= CREATININE_FILTRATE
		}
	}
	return true
}

func MuscleFindFood(ctx context.Context, cell CellActor) bool {
	request := cell.Organ().RequestWork(Work{
		workType: think,
	})
	if request.status == 200 {
		// Successfully found a food unit.
		cell.Organ().RequestWork(Work{
			workType: digest,
		})
	}
	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(Work{
		workType: cover,
	})
	return true
}

func BrainStimulateMuscles(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(Work{
		workType: move,
	})

	// Check resources in the brain. If not enough, stimulate muscle movements.
	organ := cell.Organ()
	resource := organ.materialPool.resourcePool.resources
	ligand := organ.materialPool.GetLigand()
	organ.materialPool.PutLigand(ligand)
	if resource.vitamins <= BRAIN_VITAMIN_THRESHOLD || resource.glucose <= BRAIN_GLUCOSE_THRESHOLD {
		// If glocose or vitamin levels are low, produce hunger ligands.
		ligand.hunger += 1
	} else if ligand.hunger > 1 {
		ligand.hunger -= 1
	}
	for ligand.hunger > LIGAND_HUNGER_THRESHOLD {
		fmt.Println("HUNGRY")
		cell.Organ().RequestWork(Work{
			workType: move,
		})
		ligand.hunger--
	}
	return true
}

func BrainRequestPump(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(Work{
		workType: pump,
	})
	// Check o2 and co2 in the brain. If not enough/too much, stimulate the heart.
	organ := cell.Organ()
	resource := organ.materialPool.resourcePool.resources
	waste := organ.materialPool.wastePool.wastes
	ligand := organ.materialPool.GetLigand()
	organ.materialPool.PutLigand(ligand)
	if resource.o2 <= BRAIN_O2_THRESHOLD {
		// If 02 levels are low, produce asphyxia ligands.
		ligand.asphyxia += 1
	} else if waste.co2 >= BRAIN_CO2_THRESHOLD {
		// If C02 levels are too high, produce asphyxia ligands.
		ligand.asphyxia += 1
	} else if ligand.asphyxia > 0 {
		ligand.asphyxia -= 1
	}
	for ligand.hunger > LIGAND_HUNGER_THRESHOLD {
		fmt.Println("ASPHYXIA")
		cell.Organ().RequestWork(Work{
			workType: pump,
		})
		ligand.asphyxia--
	}
	return true
}

func Flatulate(ctx context.Context, cell CellActor) bool {
	// Manage CO2 levels by leaking it.
	waste := cell.Organ().materialPool.GetWaste()
	defer cell.Organ().materialPool.PutWaste(waste)
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
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Filtrate,
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
		fallthrough
	case Podocyte:
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
	if c.CanMove() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   MoveTowardsChemotaxisCytokine,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   WillMitosisAndRepair,
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

func MoveAwayFromChemotaxisCytokineOrRandomly(ctx context.Context, cell CellActor) bool {
	if !cell.MoveAwayFromCytokine(induce_chemotaxis) {
		return MoveRandomly(ctx, cell)
	}
	return true
}

func BacteriaWillMitosis(ctx context.Context, cell CellActor) bool {
	// Bacteria will not be allowed to repair itself.
	// Conditions for bacteria mitosis, may allow runaway growth:
	// - Enough internal energy (successful calls to Oxygenate)
	// - Enough vitamin or glucose resources (not picky)
	// - Enough time has passed
	resource := cell.Organ().materialPool.GetResource()
	defer cell.Organ().materialPool.PutResource(resource)
	if resource.glucose >= GLUCOSE_COST_MITOSIS && cell.WillMitosis() {
		resource.glucose -= GLUCOSE_COST_MITOSIS
		cell.Mitosis(ctx)
	} else if resource.vitamins >= VITAMIN_COST_MITOSIS && cell.WillMitosis() {
		resource.vitamins -= VITAMIN_COST_MITOSIS
		cell.Mitosis(ctx)
	}
	return true
}

func BacteriaConsume(ctx context.Context, cell CellActor) bool {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	if cell.CollectResources(ctx) {
		cell.Oxygenate(true)
		cell.ProduceWaste()
	}
	return true
}

func BacteriaShouldApoptosis(ctx context.Context, cell CellActor) bool {
	waste := cell.Organ().materialPool.GetWaste()
	defer cell.Organ().materialPool.PutWaste(waste)
	if waste.creatinine >= CREATININE_THRESHOLD {
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
				action:   BacteriaWillMitosis,
				proteins: GenerateRandomProteinPermutation(c),
			},
		},
	}
	currNode := s.root
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   BacteriaConsume,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	currNode = currNode.next
	if c.CanMove() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   MoveAwayFromChemotaxisCytokineOrRandomly,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
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
