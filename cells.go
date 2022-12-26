package main

import (
	"container/ring"
	"context"
	"fmt"
	"image"
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
	Podocyte      // Kidney Cell
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
	case Podocyte:
		return "Podocyte"
	case TLymphocyte:
		return "TLymphocyte"
	case Dendritic:
		return "Dendritic"
	}
	return "unknown"
}

type CellActor interface {
	Worker
	AntigenPresenting
	CellType() CellType
	Start(context.Context)
	SetStop(context.CancelFunc)
	Stop()
	Organ() *Node
	Tissue() *Tissue
	Position() image.Point
	LastPositions() *ring.Ring
	Function() *StateDiagram
	WillMitosis() bool
	Mitosis(ctx context.Context)
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
	MoveToPoint(pt image.Point)
	MoveTowardsCytokine(CytokineType) bool
	MoveAwayFromCytokine(CytokineType) bool
}

type Cell struct {
	sync.RWMutex
	cellType     CellType
	dna          *DNA
	mhc_i        MHC_I
	antigen      *Antigen
	workType     WorkType
	organ        *Node
	resourceNeed *ResourceBlob
	damage       int
	function     *StateDiagram
	oxygenated   bool
	render       *Renderable
	stop         context.CancelFunc
}

func (c *Cell) String() string {
	return fmt.Sprintf("%v (%v)", c.cellType, c.dna.name)
}

func (c *Cell) SetStop(stop context.CancelFunc) {
	c.stop = stop
}

func (c *Cell) Stop() {
	c.stop()
}

func (c *Cell) CanTransport() bool {
	return false
}

func (c *Cell) Organ() *Node {
	return c.organ
}

func (c *Cell) SetOrgan(node *Node) {
	c.organ = node
}

func (c *Cell) Tissue() *Tissue {
	if c.organ == nil || c.organ.tissue == nil || c.organ.tissue.rootMatrix == nil {
		return nil
	}
	return c.organ.tissue
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
	c.damage += int(damage)
	fmt.Println("Damaged:", c, "in", c.organ)
}

func (c *Cell) CleanUp() {
	c.render.visible = false
	c.organ.tissue.Detach(c.render)
	c.Stop()
	c.organ = nil
}

func (c *Cell) Work(ctx context.Context, request Work) Work {
	if request.workType != c.workType {
		log.Fatalf("Cell %v is unable to perform work: %v", c, request)
	}
	if c.organ.materialPool != nil && !c.CollectResources(ctx) {
		if c.cellType != RedBlood {
			if c.resourceNeed.glucose > 0 {
				fmt.Println("Not enough glucose:", c, "in", c.organ)
			}
			if c.resourceNeed.o2 > 0 {
				fmt.Println("Not enough oxygen:", c, "in", c.organ)
			}
		}
		request.status = 503
		request.result = "Not enough resources."
		return request
	}
	c.Lock()
	switch c.cellType {
	case RedBlood:
		waste := c.organ.materialPool.GetWaste()
		waste.co2 += CELLULAR_TRANSPORT_CO2
		c.organ.materialPool.PutWaste(waste)
	case Enterocyte:
		resource := c.organ.materialPool.GetResource()
		resource.glucose += GLUCOSE_INTAKE
		resource.vitamins += VITAMIN_INTAKE
		c.organ.materialPool.PutResource(resource)
	case Podocyte:
		waste := c.organ.materialPool.GetWaste()
		if waste.creatinine <= CREATININE_FILTRATE {
			waste.creatinine = 0
		} else {
			waste.creatinine -= CREATININE_FILTRATE
		}
		c.organ.materialPool.PutWaste(waste)
	case Pneumocyte:
		waste := c.organ.materialPool.GetWaste()
		if waste.co2 <= CELLULAR_TRANSPORT_CO2 {
			waste.co2 = 0
		} else {
			waste.co2 -= CELLULAR_TRANSPORT_CO2
		}
		c.organ.materialPool.PutWaste(waste)
		resource := c.organ.materialPool.GetResource()
		resource.o2 += LUNG_O2_INTAKE
		c.organ.materialPool.PutResource(resource)
	case Cardiomyocyte:
		request := c.organ.RequestWork(Work{
			workType: exhale,
		})
		if request.status == 200 {
			resource := c.organ.materialPool.GetResource()
			resource.o2 += CELLULAR_TRANSPORT_O2
			c.organ.materialPool.PutResource(resource)
		}
	case Myocyte:
		fallthrough
	case Keratinocyte:
		fallthrough
	case Neuron:
		fallthrough
	default:
	}
	request.status = 200
	request.result = "Completed."
	c.ProduceWaste()
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

func (c *Cell) CollectResources(ctx context.Context) bool {
	if c.resourceNeed == nil {
		c.ResetResourceNeed()
	}
	for !reflect.DeepEqual(c.resourceNeed, new(ResourceBlob)) {
		select {
		case <-ctx.Done():
			return false
		default:
			resource := c.organ.materialPool.GetResource()
			resource.Consume(c.resourceNeed)
			c.organ.materialPool.PutResource(resource)
		}
	}
	c.ResetResourceNeed()
	return true
}

func (c *Cell) ResetResourceNeed() {
	switch c.cellType {
	case Keratinocyte:
		fallthrough
	case Enterocyte:
		fallthrough
	case Podocyte:
		c.resourceNeed = &ResourceBlob{
			o2:      0,
			glucose: 0,
		}
	case Pneumocyte:
		c.resourceNeed = &ResourceBlob{
			o2:      CELLULAR_TRANSPORT_O2,
			glucose: 0,
		}
	case RedBlood:
		c.resourceNeed = &ResourceBlob{
			o2:      CELLULAR_TRANSPORT_O2,
			glucose: CELLULAR_TRANSPORT_GLUCOSE,
		}
	case Bacteroidota:
		c.resourceNeed = &ResourceBlob{
			o2:      CELLULAR_RESPIRATION_O2,
			glucose: CELLULAR_RESPIRATION_GLUCOSE,
		}
	case Neuron:
		fallthrough
	case Cardiomyocyte:
		fallthrough
	case Myocyte:
		fallthrough
	default:
		c.resourceNeed = &ResourceBlob{
			o2:      CELLULAR_RESPIRATION_O2,
			glucose: CELLULAR_RESPIRATION_GLUCOSE,
		}
	}
}

func (c *Cell) ProduceWaste() {
	if c.organ.materialPool != nil {
		switch c.cellType {
		case Pneumocyte:
			fallthrough
		case Keratinocyte:
			fallthrough
		case Enterocyte:
			fallthrough
		case Podocyte:
			fallthrough
		case RedBlood:
			// No waste produced.
		case Bacteroidota:
			waste := c.organ.materialPool.GetWaste()
			waste.co2 += CELLULAR_RESPIRATION_CO2
			c.organ.materialPool.PutWaste(waste)
			resource := c.Organ().materialPool.GetResource()
			resource.vitamins += BACTERIA_VITAMIN_PRODUCTION
			c.Organ().materialPool.PutResource(resource)
		case Cardiomyocyte:
			fallthrough
		case Myocyte:
			fallthrough
		case Neuron:
			fallthrough
		default:
			waste := c.organ.materialPool.GetWaste()
			waste.creatinine += CREATININE_PRODUCTION
			waste.co2 += CELLULAR_RESPIRATION_CO2
			c.organ.materialPool.PutWaste(waste)
		}
	}
}

func (c *Cell) Position() image.Point {
	return c.render.position
}

func (c *Cell) LastPositions() *ring.Ring {
	return c.render.lastPositions
}

func (c *Cell) CanMove() bool {
	return false
}

func (c *Cell) Move(dx, dy, dz int) {
	if c.render == nil {
		return
	}
	c.render.targetX += dx
	c.render.targetY += dy
	c.render.targetZ += dz
	tissue := c.Tissue()
	if tissue != nil {
		tissue.Move(c.render)
	}
}

func (c *Cell) MoveToPoint(pt image.Point) {
	if c.render == nil {
		return
	}
	c.render.targetX = pt.X
	c.render.targetY = pt.Y
	tissue := c.Tissue()
	if tissue != nil {
		tissue.Move(c.render)
	}
}

func (c *Cell) GetNearestCytokines(t CytokineType) (points []image.Point, concentrations []uint8) {
	tissue := c.Tissue()
	if tissue == nil {
		return
	}
	x := c.render.position.X
	x_plus := x + CYTOKINE_SENSE_RANGE
	x_minus := x - CYTOKINE_SENSE_RANGE
	y := c.render.position.Y
	y_plus := y + CYTOKINE_SENSE_RANGE
	y_minus := y - CYTOKINE_SENSE_RANGE
	points = []image.Point{{x_minus, y_plus}, {x, y_plus}, {x_plus, y_plus}, {x_minus, y}, {x, y}, {x_plus, y}, {x_minus, y_minus}, {x, y_minus}, {x_plus, y_minus}}
	concentrations = tissue.rootMatrix.GetCytokineContentrations(points, t)
	return
}

func (c *Cell) MoveTowardsCytokine(t CytokineType) bool {
	points, concentrations := c.GetNearestCytokines(t)
	maxIndex := -1
	maxVal := uint8(0)
	for i, v := range concentrations {
		if v > maxVal {
			maxVal = v
			maxIndex = i
		}
	}
	if maxIndex >= 0 {
		c.MoveToPoint(points[maxIndex])
	}
	return maxIndex >= 0
}

func (c *Cell) MoveAwayFromCytokine(t CytokineType) bool {
	points, concentrations := c.GetNearestCytokines(t)
	minIndex := -1
	minVal := uint8(0)
	for i, v := range concentrations {
		if v < minVal {
			minVal = v
			minIndex = i
		}
	}
	if minIndex >= 0 {
		c.MoveToPoint(points[minIndex])
	}
	return minIndex >= 0
}

func (c *Cell) DropCytokine(t CytokineType, concentration uint8) uint8 {
	tissue := c.Tissue()
	if tissue != nil {
		return tissue.AddCytokine(c.render, t, concentration)
	}
	return 0
}

type EukaryoticCell struct {
	*Cell
}

func (e *EukaryoticCell) Start(ctx context.Context) {
	e.function = e.dna.makeFunction(e)
	go e.function.Run(ctx, e)
	e.Tissue().Attach(e.render)
}

func (e *EukaryoticCell) SetOrgan(node *Node) {
	e.Cell.SetOrgan(node)
	e.organ.AddWorker(e)
}

func (e *EukaryoticCell) PresentProteins() (proteins []Protein) {
	if e.function == nil {
		return
	}
	return e.function.current.function.proteins
}

func (e *EukaryoticCell) Mitosis(ctx context.Context) {
	if e.organ == nil {
		return
	}
	MakeTransportRequest(e.organ.transportUrl, e.dna.name, e.dna, e.cellType, e.workType, string(e.render.id))

}

func (e *EukaryoticCell) IncurDamage(damage int) {
	e.Cell.IncurDamage(damage)
	e.DropCytokine(cell_damage, CYTOKINE_CELL_DAMAGE)
}

func (e *EukaryoticCell) CleanUp() {
	e.Cell.CleanUp()
	if e.organ != nil {
		e.organ.RemoveWorker(e)
	}
	// TODO: make sure this gets garbage collected.
}

func (e *EukaryoticCell) Apoptosis() {
	e.CleanUp()
}

func (e *EukaryoticCell) WillMitosis() bool {
	ligand := e.Organ().materialPool.GetLigand()
	defer e.Organ().materialPool.PutLigand(ligand)
	if ligand.growth >= LIGAND_GROWTH_THRESHOLD {
		// Only checked if prior conditions are met.
		ligand.growth -= LIGAND_GROWTH_THRESHOLD
		return true
	}
	return false
}

func (e *EukaryoticCell) IsAerobic() bool {
	switch e.cellType {
	case Pneumocyte:
		fallthrough
	case Podocyte:
		fallthrough
	case Keratinocyte:
		return false
	}
	return true
}

func CopyEukaryoticStemCell(base *EukaryoticCell) *EukaryoticCell {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &EukaryoticCell{
		Cell: &Cell{
			cellType: base.cellType,
			dna:      base.dna,
			mhc_i:    base.dna.MHC_I(),
			workType: base.workType,
			render: &Renderable{
				id:            MakeRenderId(base.cellType.String()),
				visible:       true,
				position:      position,
				targetX:       base.render.targetX,
				targetY:       base.render.targetY,
				targetZ:       base.render.targetZ,
				lastPositions: positionTracker,
			},
		},
	}
}

type ProkaryoticCell struct {
	*Cell
	generationTime     time.Duration
	lastGenerationTime time.Time
	energy             int
}

func CopyProkaryoticCell(base *ProkaryoticCell) *ProkaryoticCell {
	var generationTime time.Duration
	switch base.cellType {
	case Bacteroidota:
		generationTime = GUT_BACTERIA_GENERATION_DURATION
	default:
		generationTime = DEFAULT_BACTERIA_GENERATION_DURATION
	}
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &ProkaryoticCell{
		Cell: &Cell{
			cellType: base.cellType,
			dna:      base.dna,
			mhc_i:    base.dna.MHC_I(),
			render: &Renderable{
				id:            MakeRenderId(base.cellType.String()),
				visible:       true,
				position:      position,
				targetX:       base.render.targetX,
				targetY:       base.render.targetY,
				targetZ:       base.render.targetZ,
				lastPositions: positionTracker,
			},
		},
		generationTime:     generationTime,
		lastGenerationTime: time.Now(),
	}
}

func (p *ProkaryoticCell) Start(ctx context.Context) {
	p.function = p.dna.makeFunction(p)
	go p.function.Run(ctx, p)
	p.Tissue().Attach(p.render)
}

func (p *ProkaryoticCell) CanTransport() bool {
	return true
}

func (p *ProkaryoticCell) Apoptosis() {
	p.CleanUp()
}

func (p *ProkaryoticCell) WillMitosis() bool {
	if time.Now().After(p.lastGenerationTime.Add(p.generationTime)) && p.energy >= BACTERIA_ENERGY_MITOSIS_THRESHOLD {
		p.energy = 0
		return true
	}
	return false
}

func (p *ProkaryoticCell) Mitosis(ctx context.Context) {
	p.lastGenerationTime = time.Now()
	if p.organ == nil {
		return
	}
	MakeTransportRequest(p.organ.transportUrl, p.cellType.String(), p.dna, p.cellType, 0, string(p.render.id))
}

func (p *ProkaryoticCell) Oxygenate(oxygenate bool) {
	p.Cell.Oxygenate(oxygenate)
	if oxygenate {
		p.energy++
	}
}

func (p *ProkaryoticCell) CanMove() bool {
	return true
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

type Leukocyte struct {
	Cell
}

type AntigenPresenting interface {
	PresentAntigen(reset bool) *Antigen
	SetDNA(*DNA)
	DNA() *DNA
}

func (i *Leukocyte) CanTransport() bool {
	return true
}

func (i *Leukocyte) CheckAntigen(c AntigenPresenting) bool {
	if c.PresentAntigen(false) == nil ||
		!i.dna.Verify(i.mhc_i, c.PresentAntigen(false)) {
		fmt.Println("KILL:", c)
		return false
	} else {
		fmt.Println("Passes:", c)
		return true
	}
}

func (i *Leukocyte) CanMove() bool {
	return true
}

type TCell struct {
	Leukocyte
	proteinReceptor Protein
}

func MakeTCell(dna *DNA, proteinReceptor Protein) *TCell {
	return &TCell{
		Leukocyte: Leukocyte{
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
	Leukocyte
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
		Leukocyte: Leukocyte{
			Cell: Cell{
				cellType: Dendritic,
				dna:      dna,
				mhc_i:    dna.MHC_I(),
			},
		},
		proteinSignatures: make(map[Protein]bool),
	}
}
