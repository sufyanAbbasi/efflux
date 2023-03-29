package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand"
	"sync"
	"time"
)

type StateDiagram struct {
	sync.RWMutex
	root    *StateNode
	current *StateNode
}

func (s *StateDiagram) Run(ctx context.Context, cell CellActor) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	// Cell may be killed during the execution of this function.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
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
					if hasOrganOrCleanup(ctx, cell) && s.current.function != nil {
						cell.BroadcastExistence(ctx)
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
	return nil
}

func (s *StateDiagram) GetLastNode() *StateNode {
	seen := map[*StateNode]bool{
		s.root: true,
	}
	lastNode := s.root
	for _, isLast := seen[lastNode.next]; !isLast && lastNode.next != nil; _, isLast = seen[lastNode.next] {
		seen[lastNode] = true
		lastNode = lastNode.next
	}
	return lastNode
}

func (s *StateDiagram) Graft(mutation *StateDiagram) {
	s.Lock()
	defer s.Unlock()
	cellLast := s.GetLastNode()
	mutationLast := mutation.GetLastNode()
	mutationLast.next = cellLast.next
	cellLast.next = mutation.root
}

type StateNode struct {
	next     *StateNode
	function *ProteinFunction
}

// Return false if terminal
type CellAction func(ctx context.Context, cell CellActor) bool

type ProteinFunction struct {
	action   CellAction
	proteins []Protein
}

func (p *ProteinFunction) Run(ctx context.Context, cell CellActor) bool {
	return p.action(ctx, cell)
}

// General Actions

func hasOrganOrCleanup(ctx context.Context, cell CellActor) bool {
	if cell.Organ() == nil {
		fmt.Printf("Force killed: %v\n", cell)
		defer cell.CleanUp()
		return false
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
	if !cell.MoveTowardsCytokines([]CytokineType{induce_chemotaxis}) {
		return Explore(ctx, cell)
	}
	return true
}

func MoveTowardsCellDamageCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if !cell.MoveTowardsCytokines([]CytokineType{cell_damage}) {
		return Explore(ctx, cell)
	}
	return true
}

func MoveTowardsCellStressCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if !cell.MoveTowardsCytokines([]CytokineType{cell_stressed}) {
		return Explore(ctx, cell)
	}
	return true
}

func MoveTowardsAntigenPresentCytokineOrExplore(ctx context.Context, cell CellActor) bool {
	if !cell.MoveTowardsCytokines([]CytokineType{antigen_present}) {
		return Explore(ctx, cell)
	}
	return true
}

func WillMitosisAndRepair(ctx context.Context, cell CellActor) bool {
	// Not all cells can repair, but for the sake of this simulation, they can.
	resource := cell.Organ().materialPool.GetResource(ctx)
	defer cell.Organ().materialPool.PutResource(resource)
	if cell.Damage() > 0 && cell.CanRepair() {
		repair := resource.vitamins
		if repair > MAX_REPAIR {
			repair = MAX_REPAIR
		}
		if repair > cell.Damage() {
			repair = cell.Damage()
		}
		resource.vitamins -= repair
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
		antibodyCount := 0
		antibodyLoad := cell.AntibodyLoad()
		if antibodyLoad != nil {
			antibodyCount = int(antibodyLoad.concentration)
		}
		cell.IncurDamage(1 + antibodyCount)
	}
	if cell.Damage() > MAX_DAMAGE {
		Apoptosis(ctx, cell)
		return false
	}
	return true
}

func Apoptosis(ctx context.Context, cell CellActor) bool {
	fmt.Println(cell, " Died in", cell.Organ())
	cell.Apoptosis(true)
	return false
}

func Respirate(ctx context.Context, cell CellActor) bool {
	// Receive a unit of 02 for a unit of CO2
	request := cell.Organ().RequestWork(ctx, Work{
		workType: WorkType_exchange,
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
		workType: WorkType_exhale,
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
		workType: WorkType_filter,
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
		workType: WorkType_think,
	})
	if request.status == 200 {
		// Successfully found a food unit.
		cell.Organ().RequestWork(ctx, Work{
			workType: WorkType_digest,
		})
	}
	return true
}

func MuscleSeekSkinProtection(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: WorkType_cover,
	})
	return true
}

func BrainStimulateMuscles(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: WorkType_move,
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
			workType: WorkType_move,
		})
		ligand.hunger--
	}
	return true
}

func BrainRequestPump(ctx context.Context, cell CellActor) bool {
	cell.Organ().RequestWork(ctx, Work{
		workType: WorkType_pump,
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
			workType: WorkType_pump,
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
		return !Transport(cell)
	}
	return true
}

func GenerateRandomProteinPermutation(dna *DNA) (proteins []Protein) {
	selfProteins := dna.selfProteins
	chooseN := len(selfProteins) / 3
	permutations := rand.Perm(chooseN)
	for i := 0; i < chooseN; i++ {
		proteins = append(proteins, selfProteins[permutations[i]])
	}
	return
}

func MakeStateDiagramByEukaryote(c CellActor, dna *DNA) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   WillMitosisAndRepair,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		},
	}
	currNode := s.root
	switch c.CellType() {
	case CellType_Pneumocyte:
		fallthrough
	case CellType_Keratinocyte:
		fallthrough
	case CellType_Podocyte:
		fallthrough
	case CellType_Hemocytoblast:
		fallthrough
	case CellType_Lymphoblast:
		fallthrough
	case CellType_Myeloblast:
		fallthrough
	case CellType_Monocyte:
		fallthrough
	case CellType_Macrophagocyte:
		fallthrough
	case CellType_Neutrocyte:
		fallthrough
	case CellType_NaturalKillerCell:
		fallthrough
	case CellType_VirginTLymphocyte:
		fallthrough
	case CellType_HelperTLymphocyte:
		fallthrough
	case CellType_KillerTLymphocyte:
		fallthrough
	case CellType_BLymphocyte:
		fallthrough
	case CellType_EffectorBLymphocyte:
		fallthrough
	case CellType_Dendritic:
		// Do nothing special.
	case CellType_RedBlood:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Expirate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Filtrate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	case CellType_Neuron:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BrainRequestPump,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BrainStimulateMuscles,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	case CellType_Enterocyte:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Flatulate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		for i := 3; i > 0; i-- {
			currNode = currNode.next
			// Larger demand for oxygen to supply gut bacteria.
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   Respirate,
					proteins: GenerateRandomProteinPermutation(dna),
				},
			}
		}
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   CheckVitaminLevels,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	case CellType_Myocyte:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   MuscleFindFood,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   MuscleSeekSkinProtection,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
		fallthrough
	case CellType_Cardiomyocyte:
		fallthrough
	default:
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Respirate,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	if c.CanMove() {
		switch c.CellType() {
		case CellType_VirginTLymphocyte:
			fallthrough
		case CellType_BLymphocyte:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   MoveTowardsChemotaxisCytokineOrExplore,
					proteins: GenerateRandomProteinPermutation(dna),
				},
			}
			currNode = currNode.next
		case CellType_Neutrocyte:
			fallthrough
		case CellType_Macrophagocyte:
			fallthrough
		case CellType_Dendritic:
			fallthrough
		case CellType_HelperTLymphocyte:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   MoveTowardsAntigenPresentCytokineOrExplore,
					proteins: GenerateRandomProteinPermutation(dna),
				},
			}
			currNode = currNode.next
		case CellType_NaturalKillerCell:
			fallthrough
		case CellType_KillerTLymphocyte:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   MoveTowardsCellStressCytokineOrExplore,
					proteins: GenerateRandomProteinPermutation(dna),
				},
			}
			currNode = currNode.next
		case CellType_Lymphoblast:
			fallthrough
		case CellType_Myeloblast:
			fallthrough
		case CellType_Monocyte:
			fallthrough
		case CellType_EffectorBLymphocyte:
			fallthrough
		default:
			currNode.next = &StateNode{
				function: &ProteinFunction{
					action:   Explore,
					proteins: GenerateRandomProteinPermutation(dna),
				},
			}
			currNode = currNode.next
		}
	}
	if c.CanInteract() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   Interact,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	if c.DoesWork() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   DoWork,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	if c.CanTransport() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   ShouldTransport,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: s.root, // Loop back.
		function: &ProteinFunction{
			action:   ShouldApoptosis,
			proteins: GenerateRandomProteinPermutation(dna),
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
	case CellType_Bacteroidota:
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

func MakeStateDiagramByProkaryote(c CellActor, dna *DNA) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   BacteriaWillMitosis,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		},
	}
	currNode := s.root
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   BacteriaConsume,
			proteins: GenerateRandomProteinPermutation(dna),
		},
	}
	currNode = currNode.next
	if c.CanMove() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   BacteriaMoveAwayFromCytokinesOrExplore,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	if c.CanTransport() {
		currNode.next = &StateNode{
			function: &ProteinFunction{
				action:   ShouldTransport,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		}
		currNode = currNode.next
	}
	currNode.next = &StateNode{
		next: s.root, // Back to beginning.
		function: &ProteinFunction{
			action:   ShouldApoptosis,
			proteins: GenerateRandomProteinPermutation(dna),
		},
	}
	return s
}

// Virus StateDiagrams

func MakeVirusProtein(ctx context.Context, cell CellActor) bool {
	viralLoad := cell.ViralLoad()
	if viralLoad == nil {
		return true
	}
	viralLoad.Lock()
	defer viralLoad.Unlock()
	viralLoad.concentration++
	if viralLoad.concentration%INTERFERON_PRODUCTION_MOD == 0 {
		cell.IncurDamage(int(math.Sqrt(float64(viralLoad.concentration))))
	}

	if viralLoad.concentration >= BURST_VIRUS_CONCENTRATION {
		fmt.Println(cell, "bursting with", viralLoad.virus)
		cell.IncurDamage(MAX_DAMAGE)
	}
	return true
}

func ProduceInterferon(ctx context.Context, cell CellActor) bool {
	viralLoad := cell.ViralLoad()
	if viralLoad == nil {
		return true
	}
	if viralLoad.concentration%INTERFERON_PRODUCTION_MOD == 0 {
		cell.DropCytokine(cell_stressed, CYTOKINE_CELL_STRESSED)
	}
	return true
}

func MakeStateDiagramByVirus(c CellActor, dna *DNA) *StateDiagram {
	s := &StateDiagram{
		root: &StateNode{
			function: &ProteinFunction{
				action:   MakeVirusProtein,
				proteins: GenerateRandomProteinPermutation(dna),
			},
		},
	}
	currNode := s.root
	currNode.next = &StateNode{
		function: &ProteinFunction{
			action:   ProduceInterferon,
			proteins: GenerateRandomProteinPermutation(dna),
		},
		next: s.root,
	}
	return s
}
