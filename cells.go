package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

type CellType int

const (
	Bacterial    CellType = iota
	Bacteroidota          // Bacteria that synthesize vitamins in the gut.
	RedBlood
	Neuron
	Cardiomyocyte // Heart Cell
	Pneumocyte    // Pulmonary Cell
	Myocyte       // Muscle Cell
	Keratinocyte  // Skin Cell
	Enterocyte    // Gut Lining Cell
	TLymphocyte   // T Cell
	Dendritic     // Dendritic Cells
)

func (c CellType) String() string {
	switch c {
	case Bacterial:
		return "Bacterial"
	case Bacteroidota:
		return "Bacteroidota"
	case RedBlood:
		return "RedBlood"
	case Neuron:
		return "Neuron"
	case Cardiomyocyte:
		return "Cardiomyocyte"
	case Pneumocyte:
		return "Pneumocyte"
	case Myocyte:
		return "Myocyte"
	case Keratinocyte:
		return "Keratinocyte"
	case Enterocyte:
		return "Enterocyte"
	case TLymphocyte:
		return "TLymphocyte"
	case Dendritic:
		return "Dendritic"
	}
	return "unknown"
}

type Cell struct {
	sync.RWMutex
	cellType     CellType
	dna          *DNA
	mhc_i        MHC_I
	antigen      *Antigen
	workType     WorkType
	parent       *Node
	resourceNeed *ResourceBlob
	damage       int
	function     *StateDiagram
	oxygenated   bool
}

func (c *Cell) String() string {
	return fmt.Sprintf("%v (%v)", c.cellType, c.dna.name)
}

func (c *Cell) Parent() *Node {
	return c.parent
}

func (c *Cell) SetParent(node *Node) {
	c.parent = node
}

func (c *Cell) DNA() *DNA {
	return c.dna
}

func (c *Cell) SetDNA(dna *DNA) {
	c.dna = dna
}

func (c *Cell) CellType() CellType {
	return c.cellType
}

func (c *Cell) Function() *StateDiagram {
	return c.function
}

func (c *Cell) WorkType() WorkType {
	return c.workType
}

func (c *Cell) IsOxygenated() bool {
	return c.oxygenated
}

func (c *Cell) Oxygenate(oxygenate bool) {
	c.oxygenated = oxygenate
}

func (c *Cell) Damage() int {
	return c.damage
}
func (c *Cell) Repair(damage int) {
	if c.damage <= damage {
		c.damage = 0
	} else {
		c.damage -= damage
	}
	fmt.Println("Repaired:", c)
}

func (c *Cell) IncurDamage(damage int) {
	c.damage += damage
	fmt.Println("Damaged:", c)
}

func (c *Cell) Work(ctx context.Context, request Work) Work {
	if request.workType != c.workType {
		log.Fatalf("Cell %v is unable to perform work: %v", c, request)
	}
	if c.parent.materialPool != nil && !c.CollectResources(ctx) {
		request.status = 503
		request.result = "Not enough resources."
		return request
	}
	c.Lock()
	switch c.cellType {
	case RedBlood:
		waste := c.parent.materialPool.GetWaste()
		waste.co2 += 6
		c.parent.materialPool.PutWaste(waste)
		resource := c.parent.materialPool.GetResource()
		if resource.o2 <= 6 {
			resource.o2 = 0
		} else {
			resource.o2 -= 6
		}
		c.parent.materialPool.PutResource(resource)
		request.status = 200
		request.result = "Completed."
	case Enterocyte:
		resource := c.parent.materialPool.GetResource()
		resource.glucose += 6
		c.parent.materialPool.PutResource(resource)
		request.status = 200
		request.result = "Completed."
	case Pneumocyte:
		// 6 CO2 will be removed at destination
		// 6 O2 added at destination
		fallthrough
	case Keratinocyte:
		fallthrough
	case Neuron:
		fallthrough
	case Cardiomyocyte:
		fallthrough
	case Myocyte:
		fallthrough
	default:
		request.status = 200
		request.result = "Completed."
		c.ProduceWaste()
	}
	c.Unlock()
	return request
}

func (c *Cell) SampleProteins() (proteins []Protein) {
	if c.function != nil && c.function.current != nil {
		return c.function.current.function.proteins
	}
	return
}

func (c *Cell) PresentAntigen(reset bool) *Antigen {
	if reset || c.antigen == nil {
		c.antigen = c.dna.GenerateAntigen(c.SampleProteins())
	}
	return c.antigen
}

func (c *Cell) ResetResourceNeed() {
	switch c.cellType {
	case Pneumocyte:
		fallthrough
	case Keratinocyte:
		fallthrough
	case RedBlood:
		c.resourceNeed = &ResourceBlob{
			o2: 0,
		}
	case Neuron:
		fallthrough
	case Cardiomyocyte:
		fallthrough
	case Myocyte:
		fallthrough
	default:
		c.resourceNeed = &ResourceBlob{
			o2: 6,
		}
	}
}

func (c *Cell) CollectResources(ctx context.Context) bool {
	if c.resourceNeed == nil {
		c.ResetResourceNeed()
	}
	for !reflect.DeepEqual(c.resourceNeed, new(ResourceBlob)) {
		select {
		case <-ctx.Done():
			return false
		default:
			resource := c.parent.materialPool.GetResource()
			defer c.parent.materialPool.PutResource(resource)
			resource.Consume(c.resourceNeed)
		}
	}
	c.ResetResourceNeed()
	return true
}

func (c *Cell) ProduceWaste() {
	if c.parent.materialPool != nil {
		switch c.cellType {
		case Pneumocyte:
			fallthrough
		case Keratinocyte:
			fallthrough
		case RedBlood:
			// No waste produced.
		case Neuron:
			fallthrough
		case Cardiomyocyte:
			fallthrough
		case Myocyte:
			fallthrough
		default:
			waste := c.parent.materialPool.GetWaste()
			waste.co2 += 6
			c.parent.materialPool.PutWaste(waste)
		}
	}
}

type EukaryoticCell struct {
	*Cell
	telomereLength int
	hasTelomerase  bool
}

func (e *EukaryoticCell) Start(ctx context.Context) {
	e.function = e.dna.makeFunction(e)
	go e.function.Run(ctx, e)
}

func (e *EukaryoticCell) PresentProteins() (proteins []Protein) {
	if e.function == nil {
		return
	}
	return e.function.current.function.proteins
}

func (e *EukaryoticCell) HasTelomerase() bool {
	return e.hasTelomerase
}

func (e *EukaryoticCell) Mitosis() CellActor {
	if e.telomereLength <= 0 {
		return nil
	}
	e.hasTelomerase = false
	e.telomereLength--
	cell := MakeEukaryoticStemCell(e.dna, e.cellType, e.workType)
	cell.parent = e.parent
	cell.telomereLength = e.telomereLength
	cell.hasTelomerase = false
	if e.parent != nil {
		e.parent.AddWorker(cell)
	}
	fmt.Println("Spawned:", cell, "in", cell.parent)
	return CellActor(cell)
}

func (e *EukaryoticCell) Apoptosis() {
	if e.parent != nil {
		e.parent.RemoveWorker(e)
	}
	e.parent = nil
	// TODO: make sure this gets garbage collected.
}

func (e *EukaryoticCell) ShouldMitosis() bool {
	ligand := e.Parent().materialPool.GetLigand()
	defer e.Parent().materialPool.PutLigand(ligand)
	if ligand.growth >= LIGAND_GROWTH_THRESHOLD {
		ligand.growth -= LIGAND_GROWTH_THRESHOLD
		return true
	}
	return false
}

func (e *EukaryoticCell) IsAerobic() bool {
	switch e.cellType {
	case Pneumocyte:
		fallthrough
	case Keratinocyte:
		return false
	}
	return true
}

func MakeEukaryoticStemCell(dna *DNA, cellType CellType, workType WorkType) *EukaryoticCell {
	return &EukaryoticCell{
		Cell: &Cell{
			cellType: cellType,
			dna:      dna,
			mhc_i:    dna.MHC_I(),
			workType: workType,
		},
		telomereLength: 100,
		hasTelomerase:  true,
	}
}

type ProkaryoticCell struct {
	Cell
	generationTime     time.Duration
	lastGenerationTime time.Time
	energy             int
}

func MakeProkaryoticCell(dna *DNA, cellType CellType) *ProkaryoticCell {
	var generationTime time.Duration
	switch cellType {
	case Bacteroidota:
		generationTime = GUT_BACTERIA_GENERATION_DURATION
	default:
		generationTime = DEFAULT_BACTERIA_GENERATION_DURATION
	}
	return &ProkaryoticCell{
		Cell: Cell{
			cellType: cellType,
			dna:      dna,
			mhc_i:    dna.MHC_I(),
		},
		generationTime:     generationTime,
		lastGenerationTime: time.Now(),
	}
}

func (p *ProkaryoticCell) Start(ctx context.Context) {
	p.function = p.dna.makeFunction(p)
	go p.function.Run(ctx, p)
}

func (p *ProkaryoticCell) HasTelomerase() bool {
	return false
}

func (p *ProkaryoticCell) Apoptosis() {
	p.parent = nil
	// TODO: make sure this gets garbage collected.
}

func (p *ProkaryoticCell) ShouldMitosis() bool {
	if time.Now().After(p.lastGenerationTime.Add(p.generationTime)) && p.energy >= BACTERIA_ENERGY_MITOSIS_THRESHOLD {
		p.energy = 0
		return true
	}
	return false
}

func (p *ProkaryoticCell) Mitosis() CellActor {
	p.lastGenerationTime = time.Now()
	cell := MakeProkaryoticCell(p.dna, p.cellType)
	cell.parent = p.parent
	fmt.Println("Spawned:", cell, "in", cell.parent)
	return CellActor(cell)
}

func (p *ProkaryoticCell) Oxygenate(oxygenate bool) {
	p.Cell.Oxygenate(oxygenate)
	if !oxygenate {
		p.energy++
	}
}

func (p *ProkaryoticCell) IsAerobic() bool {
	switch p.cellType {
	case Bacteroidota:
		return true
	}
	return false
}

type Virus struct {
	dna              *DNA
	cellTypeToInfect CellType
}

func MakeVirus(dna *DNA, function *StateDiagram, cellTypeToInfect CellType) *Virus {
	dna.makeFunction = ProduceVirus
	return &Virus{
		dna:              dna,
		cellTypeToInfect: cellTypeToInfect,
	}
}

func (v *Virus) InfectCell(c CellActor) {
	if c.CellType() == v.cellTypeToInfect {
		c.SetDNA(v.dna)
		c.PresentAntigen(true)
		function := c.Function()
		if function != nil {
			function.Graft(v.dna.makeFunction(c))
		}
	}
}

type ImmuneCell struct {
	Cell
}

type AntigenPresenting interface {
	PresentAntigen(reset bool) *Antigen
	SetDNA(*DNA)
	DNA() *DNA
}

func (i *ImmuneCell) CheckAntigen(c AntigenPresenting) bool {
	if c.PresentAntigen(false) == nil ||
		!i.dna.Verify(i.mhc_i, c.PresentAntigen(false)) {
		fmt.Println("KILL:", c)
		return false
	} else {
		fmt.Println("Passes:", c)
		return true
	}
}

type TCell struct {
	ImmuneCell
	proteinReceptor Protein
}

func MakeTCell(dna *DNA, proteinReceptor Protein) *TCell {
	return &TCell{
		ImmuneCell: ImmuneCell{
			Cell: Cell{
				cellType: TLymphocyte,
				dna:      dna,
				mhc_i:    dna.MHC_I(),
			},
		},
		proteinReceptor: proteinReceptor,
	}
}

func GenerateTCells(dna *DNA) (tCells []*TCell) {
	for i := 0; i < 65535; i++ {
		_, isSelf := dna.selfProteins[Protein(i)]
		if !isSelf {
			tCells = append(tCells, MakeTCell(dna, Protein(i)))
		}
	}
	return
}

type DendriticCell struct {
	ImmuneCell
	proteinSignatures map[Protein]bool
}

func (d *DendriticCell) Collect(t AntigenPresenting) {
	for _, p := range t.PresentAntigen(false).proteins {
		d.proteinSignatures[p] = false
	}
}

func (d *DendriticCell) FoundMatch(t *TCell) bool {
	_, found := d.proteinSignatures[t.proteinReceptor]
	if found {
		d.proteinSignatures[t.proteinReceptor] = found
	}
	return found
}

func MakeDendriticCell(dna *DNA) *DendriticCell {
	return &DendriticCell{
		ImmuneCell: ImmuneCell{
			Cell: Cell{
				cellType: Dendritic,
				dna:      dna,
				mhc_i:    dna.MHC_I(),
			},
		},
		proteinSignatures: make(map[Protein]bool),
	}
}

func MakeCellFromRequest(request TransportRequest) (CellActor, error) {
	dna, err := MakeDNAFromRequest(request)
	if err != nil {
		return nil, err
	}
	var cell CellActor
	switch request.CellType {
	case Bacterial:
		fallthrough
	case Bacteroidota:
		cell = MakeProkaryoticCell(dna, request.CellType)
	case RedBlood:
		fallthrough
	case Neuron:
		fallthrough
	case Cardiomyocyte:
		fallthrough
	case Pneumocyte:
		fallthrough
	case Myocyte:
		fallthrough
	case Keratinocyte:
		fallthrough
	case Enterocyte:
		fallthrough
	case TLymphocyte:
		fallthrough
	case Dendritic:
		fallthrough
	default:
		cell = MakeEukaryoticStemCell(dna, request.CellType, request.WorkType)
	}
	return cell, nil
}
