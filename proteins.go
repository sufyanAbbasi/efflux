package main

import (
	"context"
	"fmt"
	"image"
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
				cell.CleanUp()
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
			default:
				cell.BroadcastPosition(ctx)
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
	cell.DoWork(ctx)
	return true
}

func Explore(ctx context.Context, cell CellActor) bool {
	tissue := cell.Tissue()
	if tissue == nil {
		return true
	}
	p := cell.Position()
	x := p.X
	x_plus := x + 1
	x_minus := x - 1
	y := p.Y
	y_plus := y + 1
	y_minus := y - 1
	points := []image.Point{
		{x_minus, y_plus},
		{x, y_plus},
		{x_plus, y_plus},
		{x_minus, y},
		{x_plus, y},
		{x_minus, y_minus},
		{x, y_minus},
		{x_plus, y_minus},
	}
	isOpen := tissue.rootMatrix.GetOpenSpaces(points)
	if len(isOpen) > 0 {
		moveToPoint := isOpen[rand.Intn(len(isOpen))]
		newPositions := cell.GetUnvisitedPositions(isOpen)
		if len(newPositions) > 0 {
			moveToPoint = newPositions[rand.Intn(len(newPositions))]
		}
		cell.MoveToPoint(moveToPoint)
	} else {
		cell.Move(1-rand.Intn(3), 1-rand.Intn(3), 0)
	}
	return true
}

func MoveTowardsChemotaxisCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if rand.Intn(5) == 0 {
		return Explore(ctx, cell)
	} else if !cell.MoveTowardsCytokines([]CytokineType{induce_chemotaxis}) {
		return Explore(ctx, cell)
	}
	return true
}

func MoveTowardsCellDamageCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if rand.Intn(5) == 0 {
		return Explore(ctx, cell)
	} else if !cell.MoveTowardsCytokines([]CytokineType{cell_damage}) {
		return Explore(ctx, cell)
	}
	return true
}

func MoveTowardsAntigenPresentCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if rand.Intn(5) == 0 {
		return Explore(ctx, cell)
	} else if !cell.MoveTowardsCytokines([]CytokineType{antigen_present}) {
		return Explore(ctx, cell)
	}
	return true
}

func WillMitosisAndRepair(ctx context.Context, cell CellActor) bool {
	// Not all cells can repair, but for the sake of this simulation, they can.
	resource := cell.Organ().materialPool.GetResource(ctx)
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
		cell.WillMitosis(ctx) {
		resource.vitamins -= VITAMIN_COST_MITOSIS
		return cell.Mitosis(ctx)
	}
	return true
}

func ShouldApoptosis(ctx context.Context, cell CellActor) bool {
	if cell.ShouldIncurDamage(ctx) {
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
	request := cell.Organ().RequestWork(ctx, Work{
		workType: exchange,
	})
	if request.status == 200 {
		cell.Organ().materialPool.PutResource(&ResourceBlob{
			o2:      CELLULAR_TRANSPORT_O2,
			glucose: CELLULAR_TRANSPORT_GLUCOSE,
		})
		waste := cell.Organ().materialPool.GetWaste(ctx)
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
	request := cell.Organ().RequestWork(ctx, Work{
		workType: exhale,
	})
	if request.status == 200 {
		waste := cell.Organ().materialPool.GetWaste(ctx)
		defer cell.Organ().materialPool.PutWaste(waste)
		if waste.co2 <= CELLULAR_TRANSPORT_CO2 {
			waste.co2 = 0
		} else {
			waste.co2 -= CELLULAR_TRANSPORT_CO2
		}
		cell.Organ().materialPool.PutResource(&ResourceBlob{
			o2: CELLULAR_TRANSPORT_O2,
		})
	}
	return true
}

func Filtrate(ctx context.Context, cell CellActor) bool {
	// Remove some amount of creatinine.
	request := cell.Organ().RequestWork(ctx, Work{
		workType: filter,
	})
	if request.status == 200 {
		waste := cell.Organ().materialPool.GetWaste(ctx)
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
	request := cell.Organ().RequestWork(ctx, Work{
		workType: think,
	})
	if request.status == 200 {
		// Successfully found a food unit.
		cell.Organ().RequestWork(ctx, Work{
			workType: digest,
		})
	}
	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: cover,
	})
	return true
}

func BrainStimulateMuscles(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: move,
	})

	// Check resources in the brain. If not enough, stimulate muscle movements.
	organ := cell.Organ()
	resource := organ.materialPool.GetResource(ctx)
	defer organ.materialPool.PutResource(resource)
	ligand := organ.materialPool.GetLigand(ctx)
	defer organ.materialPool.PutLigand(ligand)
	if resource.vitamins <= BRAIN_VITAMIN_THRESHOLD || resource.glucose <= BRAIN_GLUCOSE_THRESHOLD {
		// If glocose or vitamin levels are low, produce hunger ligands.
		ligand.hunger += 1
	} else if ligand.hunger > 1 {
		ligand.hunger -= 1
	}
	for ligand.hunger > LIGAND_HUNGER_THRESHOLD {
		cell.Organ().RequestWork(ctx, Work{
			workType: move,
		})
		ligand.hunger--
	}
	return true
}

func BrainRequestPump(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: pump,
	})
	// Check o2 and co2 in the brain. If not enough/too much, stimulate the heart.
	organ := cell.Organ()
	resource := organ.materialPool.GetResource(ctx)
	defer organ.materialPool.PutResource(resource)
	waste := organ.materialPool.GetWaste(ctx)
	defer organ.materialPool.PutWaste(waste)
	ligand := organ.materialPool.GetLigand(ctx)
	defer organ.materialPool.PutLigand(ligand)
	if resource.o2 <= BRAIN_O2_THRESHOLD {
		// If 02 levels are low, produce asphyxia ligands.
		ligand.asphyxia += 1
	} else if waste.co2 >= BRAIN_CO2_THRESHOLD {
		// If C02 levels are too high, produce asphyxia ligands.
		ligand.asphyxia += 1
	} else if ligand.asphyxia > 0 {
		ligand.asphyxia -= 1
	}
	for ligand.asphyxia > LIGAND_ASPHYXIA_THRESHOLD {
		cell.Organ().RequestWork(ctx, Work{
			workType: pump,
		})
		ligand.asphyxia--
	}
	return true
}

func CheckVitaminLevels(ctx context.Context, cell CellActor) bool {
	// Add hunger ligand if vitamins are low.
	resource := cell.Organ().materialPool.GetResource(ctx)
	defer cell.Organ().materialPool.PutResource(resource)
	ligand := &LigandBlob{}
	if resource.vitamins < BRAIN_VITAMIN_THRESHOLD {
		defer cell.Organ().materialPool.PutLigand(ligand)
		ligand.hunger += 1
	}
	return true
}

func Flatulate(ctx context.Context, cell CellActor) bool {
	// Manage CO2 levels by leaking it.
	waste := cell.Organ().materialPool.GetWaste(ctx)
	defer cell.Organ().materialPool.PutWaste(waste)
	waste.co2 = 0
	return true
}

func Interact(ctx context.Context, cell CellActor) bool {
	interactions := cell.GetInteractions(ctx)
	for _, interaction := range interactions {
		if cell != interaction {
			cell.Interact(ctx, interaction)
		}
	}
	return true
}

func ShouldTransport(ctx context.Context, cell CellActor) bool {
	if cell.ShouldTransport(ctx) {
		return !cell.Transport()
	}
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
				action:   WillMitosisAndRepair,
				proteins: GenerateRandomProteinPermutation(c),
			},
		},
	}
	currNode := s.root
	switch c.CellType() {
	case Pneumocyte:
		fallthrough
	case Keratinocyte:
		fallthrough
	case Podocyte:
		fallthrough
	case Hemocytoblast:
		fallthrough
	case Lymphoblast:
		fallthrough
	case Myeloblast:
		fallthrough
	case Monoblast:
		fallthrough
	case Macrophagocyte:
		fallthrough
	case Neutrocyte:
		fallthrough
	case NaturalKillerCell:
		fallthrough
	case TLymphocyte:
		fallthrough
	case Dendritic:
		// Do nothing special.
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
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   CheckVitaminLevels,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
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
		switch c.CellType() {
		case Lymphoblast:
			fallthrough
		case Myeloblast:
			fallthrough
		case Monoblast:
			fallthrough
		case Neutrocyte:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   MoveTowardsChemotaxisCytokineOrExplore,
					proteins: GenerateRandomProteinPermutation(c),
				},
			}
			currNode = currNode.next
		case Macrophagocyte:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   MoveTowardsAntigenPresentCytokineOrExplore,
					proteins: GenerateRandomProteinPermutation(c),
				},
			}
			currNode = currNode.next
		default:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   Explore,
					proteins: GenerateRandomProteinPermutation(c),
				},
			}
			currNode = currNode.next
		}
	}
	if c.CanInteract() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Interact,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	if c.DoesWork() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   DoWork,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	if c.CanTransport() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   ShouldTransport,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: s.root, // Loop back.
		function: &ProteinFunction{
			action:   ShouldApoptosis,
			proteins: GenerateRandomProteinPermutation(c),
		},
	}
	return s
}

// Bacteria Related CellActions

func BacteriaMoveAwayFromCytokinesOrExplore(ctx context.Context, cell CellActor) bool {
	if rand.Intn(5) == 0 {
		return Explore(ctx, cell)
	} else if !cell.MoveAwayFromCytokines([]CytokineType{cytotoxins}) {
		return Explore(ctx, cell)
	}
	return true
}

func BacteriaWillMitosis(ctx context.Context, cell CellActor) bool {
	// Bacteria will not be allowed to repair itself.
	// Conditions for bacteria mitosis, may allow runaway growth:
	// - Enough internal energy (successful calls to Oxygenate)
	// - Enough vitamin or glucose resources (not picky)
	// - Enough time has passed
	resource := cell.Organ().materialPool.GetResource(ctx)
	defer cell.Organ().materialPool.PutResource(resource)
	switch cell.CellType() {
	case Bacteroidota:
		// Grow via HUNGER ligand.
		ligand := cell.Organ().materialPool.GetLigand(ctx)
		defer cell.Organ().materialPool.PutLigand(ligand)
		if ligand.hunger < LIGAND_HUNGER_THRESHOLD {
			break
		} else {
			ligand.hunger -= LIGAND_HUNGER_THRESHOLD
		}
		fallthrough
	default:
		if resource.glucose >= GLUCOSE_COST_MITOSIS && cell.WillMitosis(ctx) {
			resource.glucose -= GLUCOSE_COST_MITOSIS
			return cell.Mitosis(ctx)
		} else if resource.vitamins >= VITAMIN_COST_MITOSIS && cell.WillMitosis(ctx) {
			resource.vitamins -= VITAMIN_COST_MITOSIS
			return cell.Mitosis(ctx)
		}
	}
	return true
}

func BacteriaConsume(ctx context.Context, cell CellActor) bool {
	if cell.CollectResources(ctx) {
		cell.Oxygenate(true)
		cell.ProduceWaste()
	}
	return true
}

func BacteriaShouldTransport(ctx context.Context, cell CellActor) bool {
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
				action:   BacteriaMoveAwayFromCytokinesOrExplore,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	if c.CanTransport() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BacteriaShouldTransport,
				proteins: GenerateRandomProteinPermutation(c),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: s.root, // Back to beginning.
		function: &ProteinFunction{
			action:   ShouldApoptosis,
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
