package main

import (
	"container/ring"
	"context"
	"fmt"
	"image"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

type CellType int

const (
	Bacteria     CellType = iota
	Bacteroidota          // Bacteria that synthesize vitamins in the gut.
	RedBlood
	Neuron
	Cardiomyocyte            // Heart Cell
	Pneumocyte               // Pulmonary Cell
	Myocyte                  // Muscle Cell
	Keratinocyte             // Skin Cell
	Enterocyte               // Gut Lining Cell
	Podocyte                 // Kidney Cell
	Hemocytoblast            // Bone Marrow Stem Cell, spawns Lymphoblast and Myeloblast
	Lymphoblast              // Stem Cell, becomes NK, B cells, T cells
	Myeloblast               // Stem Cell, becomes Neutrophil, Macrophages, and Dendritic cells
	Macrophagocyte           // Macrophage
	Neutrocytes              // Neutrophils
	LargeGranularLymphocytes // Natural Killer Cells
	TLymphocyte              // T Cell
	Dendritic                // Dendritic Cells
)

func (c CellType) String() string {
	switch c {
	case Bacteria:
		return "Bacteria"
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
	case Hemocytoblast:
		return "Hemocytoblast"
	case Lymphoblast:
		return "Lymphoblast"
	case Myeloblast:
		return "Myeloblast"
	case Macrophagocyte:
		return "Macrophage"
	case LargeGranularLymphocytes:
		return "Neutrophil"
	case TLymphocyte:
		return "TLymphocyte"
	case Dendritic:
		return "Dendritic"
	}
	return "Unknown"
}

type CellActor interface {
	Worker
	AntigenPresenting
	CellType() CellType
	Start(context.Context)
	SetStop(context.CancelFunc)
	CleanUp()
	Stop()
	Organ() *Node
	Verbose() bool
	Tissue() *Tissue
	DoesWork() bool
	Position() image.Point
	LastPositions() *ring.Ring
	BroadcastPosition(ctx context.Context)
	SpawnTime() time.Time
	Function() *StateDiagram
	WillMitosis(context.Context) bool
	Mitosis(ctx context.Context) bool
	CollectResources(context.Context) bool
	ProduceWaste()
	Damage() int
	Repair(int)
	ShouldIncurDamage(context.Context) bool
	IncurDamage(int)
	Apoptosis()
	IsApoptosis() bool
	IsAerobic() bool
	IsOxygenated() bool
	Oxygenate(bool)
	CanMove() bool
	Move(dx, dy, dz int)
	MoveToPoint(pt image.Point)
	MoveTowardsCytokines([]CytokineType) bool
	MoveAwayFromCytokines([]CytokineType) bool
	CanInteract() bool
	GetInteractions(ctx context.Context) (interactions []CellActor)
	Interact(ctx context.Context, c CellActor)
	CanTransport() bool
	ShouldTransport(context.Context) bool
	Transport() bool
	RecordTransport()
}

type Cell struct {
	sync.RWMutex
	cellType      CellType
	dna           *DNA
	mhc_i         MHC_I
	antigen       *Antigen
	workType      WorkType
	organ         *Node
	resourceNeed  *ResourceBlob
	damage        int
	function      *StateDiagram
	oxygenated    bool
	render        *Renderable
	stop          context.CancelFunc
	transportPath [10]string
	wantPath      [10]string
	spawnTime     time.Time
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

func (c *Cell) Organ() *Node {
	return c.organ
}

func (c *Cell) Verbose() bool {
	return c.organ != nil && c.organ.verbose
}

func (c *Cell) SetOrgan(node *Node) {
	c.organ = node
}

func (c *Cell) Tissue() *Tissue {
	if c.organ == nil ||
		c.organ.tissue == nil ||
		c.organ.tissue.rootMatrix == nil {
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
	if c.Verbose() {
		fmt.Println("Repaired:", c)
	}
}

func (c *Cell) ShouldIncurDamage(ctx context.Context) bool {
	waste := c.Organ().materialPool.GetWaste(ctx)
	c.Organ().materialPool.PutWaste(waste)
	return waste.creatinine >= DAMAGE_CREATININE_THRESHOLD ||
		waste.co2 >= DAMAGE_CO2_THRESHOLD ||
		c.GetCytokineConcentrationAt(cytotoxins, c.Position()) > 0
}

func (c *Cell) IncurDamage(damage int) {
	c.damage += int(damage)
	if c.Verbose() {
		fmt.Println("Damaged:", c, "- damage", c.damage, "in", c.organ)
	}
}

func (c *Cell) CleanUp() {
	c.render.visible = false
	if c.organ != nil && c.organ.tissue != nil {
		c.organ.tissue.Detach(c.render)
	}
	c.Stop()
	if c.Verbose() {
		fmt.Println("Despawned:", c, "in", c.organ)
	}
	c.organ = nil
}

func (c *Cell) Apoptosis() {
	if c.Verbose() {
		fmt.Println("Apoptosis:", c, "in", c.organ)
	}
	c.CleanUp()
}

func (c *Cell) IsApoptosis() bool {
	return c.Tissue() == nil
}

func (c *Cell) Work(ctx context.Context, request Work) Work {
	if request.workType != c.workType {
		log.Fatalf("Cell %v is unable to perform work: %v", c, request)
	}
	if c.organ.materialPool != nil && !c.CollectResources(ctx) {
		if c.resourceNeed.glucose > 0 && c.Verbose() {
			fmt.Println("Not enough glucose:", c, "in", c.organ)
		}
		if c.resourceNeed.o2 > 0 && c.Verbose() {
			fmt.Println("Not enough oxygen:", c, "in", c.organ)
		}
		request.status = 503
		request.result = "Not enough resources."
		return request
	}
	c.Lock()
	switch c.cellType {
	case RedBlood:
		c.organ.materialPool.PutWaste(&WasteBlob{
			co2: CELLULAR_TRANSPORT_CO2,
		})
	case Enterocyte:
		c.organ.materialPool.PutResource(&ResourceBlob{
			glucose:  GLUCOSE_INTAKE,
			vitamins: VITAMIN_INTAKE,
		})
	case Podocyte:
		waste := c.organ.materialPool.GetWaste(ctx)
		if waste.creatinine <= CREATININE_FILTRATE {
			waste.creatinine = 0
		} else {
			waste.creatinine -= CREATININE_FILTRATE
		}
		c.organ.materialPool.PutWaste(waste)
	case Pneumocyte:
		waste := c.organ.materialPool.GetWaste(ctx)
		if waste.co2 <= CELLULAR_TRANSPORT_CO2 {
			waste.co2 = 0
		} else {
			waste.co2 -= CELLULAR_TRANSPORT_CO2
		}
		c.organ.materialPool.PutWaste(waste)
		c.organ.materialPool.PutResource(&ResourceBlob{
			o2: LUNG_O2_INTAKE,
		})
	case Cardiomyocyte:
		request := c.organ.RequestWork(ctx, Work{
			workType: exhale,
		})
		if request.status == 200 {
			c.organ.materialPool.PutResource(&ResourceBlob{
				o2: CELLULAR_TRANSPORT_O2,
			})
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
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT_SEC)
	defer cancel()
	if c.resourceNeed == nil {
		c.ResetResourceNeed()
	}
	for !reflect.DeepEqual(c.resourceNeed, new(ResourceBlob)) {
		select {
		case <-ctx.Done():
			if c.Verbose() {
				fmt.Println(c, "resource collection timeout in", c.organ)
			}
			return false
		default:
			resource := c.organ.materialPool.GetResource(ctx)
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
		fallthrough
	case Hemocytoblast:
		fallthrough
	case Lymphoblast:
		fallthrough
	case Myeloblast:
		fallthrough
	case Macrophagocyte:
		fallthrough
	case Neutrocytes:
		fallthrough
	case LargeGranularLymphocytes:
		fallthrough
	case TLymphocyte:
		fallthrough
	case Dendritic:
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
	case Bacteria:
		fallthrough
	case Bacteroidota:
		fallthrough
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
			fallthrough
		case Hemocytoblast:
			fallthrough
		case Lymphoblast:
			fallthrough
		case Myeloblast:
			fallthrough
		case Macrophagocyte:
			fallthrough
		case Neutrocytes:
			fallthrough
		case LargeGranularLymphocytes:
			fallthrough
		case TLymphocyte:
			fallthrough
		case Dendritic:
			// No waste produced.
		case Bacteroidota:
			c.organ.materialPool.PutWaste(&WasteBlob{
				co2: CELLULAR_RESPIRATION_CO2,
			})
			c.Organ().materialPool.PutResource(&ResourceBlob{
				vitamins: BACTERIA_VITAMIN_PRODUCTION,
			})
		case Bacteria:
			c.DropCytokine(cytotoxins, CYTOKINE_CYTOTOXINS)
			fallthrough
		case Cardiomyocyte:
			fallthrough
		case Myocyte:
			fallthrough
		case Neuron:
			fallthrough
		default:
			c.organ.materialPool.PutWaste(&WasteBlob{
				creatinine: CREATININE_PRODUCTION,
				co2:        CELLULAR_RESPIRATION_CO2,
			})
		}
	}
}

func (c *Cell) Position() image.Point {
	return c.render.position
}

func (c *Cell) LastPositions() *ring.Ring {
	return c.render.lastPositions
}

func (c *Cell) SpawnTime() time.Time {
	return c.spawnTime
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

func (c *Cell) GetCytokineConcentrationAt(t CytokineType, pt image.Point) uint8 {
	tissue := c.Tissue()
	if tissue == nil {
		return 0
	}
	return tissue.rootMatrix.GetCytokineContentrations([]image.Point{pt}, []CytokineType{t})[0][0]
}

func (c *Cell) GetNearestCytokines(t []CytokineType) (points []image.Point, concentrations [][]uint8) {
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

func (c *Cell) MoveTowardsCytokines(t []CytokineType) bool {
	points, concentrations := c.GetNearestCytokines(t)
	maxIndex := -1
	maxVal := uint8(0)
	for i, cns := range concentrations {
		for _, v := range cns {
			if v > maxVal {
				maxVal = v
				maxIndex = i
			}
		}
	}
	if maxIndex >= 0 {
		c.MoveToPoint(points[maxIndex])
	}
	return maxIndex >= 0
}

func (c *Cell) MoveAwayFromCytokines(t []CytokineType) bool {
	points, concentrations := c.GetNearestCytokines(t)
	minIndex := -1
	minVal := uint8(0)
	for i, cns := range concentrations {
		for _, v := range cns {
			if v < minVal {
				minVal = v
				minIndex = i
			}
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

func (c *Cell) CanInteract() bool {
	return false
}

func (c *Cell) Interact(context.Context, CellActor) {
	panic("unimplemented")
}

func (c *Cell) GetInteractions(ctx context.Context) (interactions []CellActor) {
	tissue := c.Tissue()
	if tissue == nil {
		return
	}
	interactions = tissue.GetInteractions(ctx, c.render)
	return
}

func (c *Cell) CanTransport() bool {
	return false
}

func (c *Cell) ShouldTransport(context.Context) bool {
	return false
}

func (c *Cell) Transport() bool {
	o := c.Organ()
	if o == nil {
		return false
	}
	var transportEdges []*Edge
	for _, e := range o.edges {
		switch e.edgeType {
		case blood_brain_barrier:
			fallthrough
		case neuronal:
			// Pass
		default:
			transportEdges = append(transportEdges, e)
		}
	}
	if len(transportEdges) == 0 {
		return false
	}
	var edge *Edge
	// Pick an edge from the want path if it exists, starting from the end.
	foundIndex := -1
	found := false
	for i := len(c.wantPath) - 1; i >= 0 && !found; i-- {
		for _, e := range transportEdges {
			if c.wantPath[i] == e.transportUrl {
				found = true
				foundIndex = i
			}
		}
	}
	if foundIndex >= 0 {
		edge = transportEdges[foundIndex]
	} else {
		// Pick a random, valid edge to transport to.
		edge = transportEdges[rand.Intn(len(transportEdges))]
		hasBeen := false
		for i := 0; i < len(c.transportPath) && !hasBeen; i++ {
			if edge.transportUrl == c.transportPath[i] {
				hasBeen = true
			}
		}
		if hasBeen {
			// If we've been to this edge before, reroll.
			edge = transportEdges[rand.Intn(len(transportEdges))]
		}
	}
	err := MakeTransportRequest(edge.transportUrl, c.dna.name, c.dna, c.cellType, c.workType, string(c.render.id), c.transportPath, c.wantPath)
	if err != nil {
		fmt.Printf("Unable to transport to %v: %v\n", edge.transportUrl, err)
		return false
	}
	return true
}

func (c *Cell) RecordTransport() {
	currentUrl := c.organ.transportUrl

	var transportPath [10]string
	copy(transportPath[0:], c.transportPath[1:])
	transportPath[9] = currentUrl
	c.transportPath = transportPath

	// Find the last found index of this location and truncate the want path up to it.
	var wantPath [10]string
	endIndex := len(c.wantPath) - 1
	found := false
	for i := endIndex; i >= 0 && !found; i-- {
		if c.wantPath[i] == currentUrl {
			found = true
			endIndex = i
		}
	}
	copy(wantPath[len(wantPath)-endIndex:], c.wantPath[0:endIndex])
	c.wantPath = wantPath
}

type EukaryoticCell struct {
	*Cell
}

func (e *EukaryoticCell) Start(ctx context.Context) {
	e.function = e.dna.makeFunction(e)
	go e.function.Run(ctx, e)
	e.Tissue().Attach(e.render)
}

func (e *EukaryoticCell) DoesWork() bool {
	return e.cellType != Hemocytoblast
}

func (e *EukaryoticCell) SetOrgan(node *Node) {
	e.Cell.SetOrgan(node)
	e.organ.AddWorker(e)
}

func (e *EukaryoticCell) BroadcastPosition(ctx context.Context) {
	e.Tissue().BroadcastPosition(ctx, e, e.render)
}

func (e *EukaryoticCell) PresentProteins() (proteins []Protein) {
	if e.function == nil {
		return
	}
	return e.function.current.function.proteins
}

func (e *EukaryoticCell) Mitosis(ctx context.Context) bool {
	if e.organ == nil {
		return false
	}
	MakeTransportRequest(e.organ.transportUrl, e.dna.name, e.dna, e.cellType, e.workType, string(e.render.id), e.transportPath, e.wantPath)
	return true
}

func (e *EukaryoticCell) IncurDamage(damage int) {
	e.Cell.IncurDamage(damage)
	e.DropCytokine(cell_damage, CYTOKINE_CELL_DAMAGE)
	e.Organ().materialPool.PutLigand(&LigandBlob{
		inflammation: LIGAND_INFLAMMATION_CELL_DAMAGE,
	})
}

func (e *EukaryoticCell) CleanUp() {
	e.Cell.CleanUp()
	if e.organ != nil {
		e.organ.RemoveWorker(e)
	}
	// TODO: make sure this gets garbage collected.
}

func (e *EukaryoticCell) WillMitosis(ctx context.Context) bool {
	switch e.cellType {
	case Hemocytoblast:
		// Bootstrap the mitosis function to spawn leukocyte stem cells, but only in the present of the hormone.
		hormone := e.organ.materialPool.GetHormone(ctx)
		if hormone.granulocyte_csf >= HORMONE_CSF_THRESHOLD {
			hormone.granulocyte_csf -= HORMONE_CSF_THRESHOLD
			MakeTransportRequest(e.organ.transportUrl, e.dna.name, e.dna, Myeloblast, nothing, string(e.render.id), e.transportPath, e.wantPath)
		}
		if hormone.macrophage_csf >= HORMONE_M_CSF_THRESHOLD {
			hormone.macrophage_csf -= HORMONE_M_CSF_THRESHOLD
			MakeTransportRequest(e.organ.transportUrl, e.dna.name, e.dna, Lymphoblast, nothing, string(e.render.id), e.transportPath, e.wantPath)
		}
		e.organ.materialPool.PutHormone(hormone)
		return hormone.granulocyte_csf >= HORMONE_CSF_THRESHOLD || hormone.macrophage_csf >= HORMONE_M_CSF_THRESHOLD
	default:
		ligand := e.Organ().materialPool.GetLigand(ctx)
		defer e.Organ().materialPool.PutLigand(ligand)
		if ligand.growth >= LIGAND_GROWTH_THRESHOLD {
			// Only checked if prior conditions are met.
			ligand.growth -= LIGAND_GROWTH_THRESHOLD
			return true
		}
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

func CopyEukaryoticCell(base *EukaryoticCell) *EukaryoticCell {
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
			transportPath: base.transportPath,
		},
	}
}

type Leukocyte struct {
	*Cell
	lifeSpan time.Duration
}

type AntigenPresenting interface {
	PresentAntigen(reset bool) *Antigen
	SetDNA(*DNA)
	DNA() *DNA
}

func (i *Leukocyte) CanTransport() bool {
	switch i.cellType {
	case Lymphoblast:
		return true
	case Myeloblast:
		return true
	case Macrophagocyte:
		return false
	case Neutrocytes:
		return false
	case LargeGranularLymphocytes:
		return false
	case TLymphocyte:
		return true
	case Dendritic:
		return true
	default:
		return false
	}
}

func (i Leukocyte) ShouldTransport(ctx context.Context) bool {
	switch i.cellType {
	case Lymphoblast:
		fallthrough
	case Myeloblast:
		fallthrough
	case Neutrocytes:
		// Wait until the lifespan is over before moving on.
		if time.Now().Before(i.spawnTime.Add(i.lifeSpan)) {
			return false
		}
		// If there is no inflammation, move on.
		ligand := i.organ.materialPool.GetLigand(ctx)
		defer i.organ.materialPool.PutLigand(ligand)
		if ligand.inflammation < LEUKOCYTE_INFLAMMATION_THRESHOLD {
			return true
		}
	}
	return false
}

func (i *Leukocyte) CheckAntigen(c AntigenPresenting) bool {
	if c.PresentAntigen(false) == nil ||
		!i.dna.Verify(i.mhc_i, c.PresentAntigen(false)) {
		return false
	} else {
		return true
	}
}

func (i *Leukocyte) Trap(c CellActor) {
	c.MoveToPoint(i.Position())
}

func (i *Leukocyte) DoesWork() bool {
	return false
}

func (i *Leukocyte) IsAerobic() bool {
	return false
}

func (i *Leukocyte) CanMove() bool {
	return true
}

func (i *Leukocyte) CanInteract() bool {
	return true
}

func (i *Leukocyte) Mitosis(ctx context.Context) bool {
	// https://en.wikipedia.org/wiki/Metamyelocyte#/media/File:Hematopoiesis_(human)_diagram_en.svg
	switch i.cellType {
	case Lymphoblast:
		// Can differentiate into Natural Killer, B Cell, and T Cells.
		MakeTransportRequest(i.organ.transportUrl, i.dna.name, i.dna, LargeGranularLymphocytes, nothing, string(i.render.id), i.transportPath, i.wantPath)

	case Myeloblast:
		// Can differentiate into Neutrophil, Macrophage, and Dendritic.
		MakeTransportRequest(i.organ.transportUrl, i.dna.name, i.dna, Neutrocytes, nothing, string(i.render.id), i.transportPath, i.wantPath)
	}
	// After differentiating, the existing cell will be converted to another, so clean up the existing one.
	return false
}

func (i *Leukocyte) WillMitosis(ctx context.Context) bool {
	// Overloading mitosis with differentiation. Once differentiated, the existing cell will disappear.
	switch i.cellType {
	case Lymphoblast:
		fallthrough
	case Myeloblast:
		// If there is no inflammation, don't differentiate.
		ligand := i.organ.materialPool.GetLigand(ctx)
		defer i.organ.materialPool.PutLigand(ligand)
		return ligand.inflammation >= LEUKOCYTE_INFLAMMATION_THRESHOLD
	}
	return false
}

type LeukocyteStemCell struct {
	*Leukocyte
}

func (l *LeukocyteStemCell) CanInteract() bool {
	return false
}

func (l *LeukocyteStemCell) Start(ctx context.Context) {
	l.function = l.dna.makeFunction(l)
	go l.function.Run(ctx, l)
	l.Tissue().Attach(l.render)
}

func (l *LeukocyteStemCell) BroadcastPosition(ctx context.Context) {
	l.Tissue().BroadcastPosition(ctx, l, l.render)
}

func CopyLeukocyteStemCell(base *LeukocyteStemCell) *LeukocyteStemCell {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &LeukocyteStemCell{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
	}
}

type Neutrophil struct {
	inNETosis bool
	*Leukocyte
}

func (n *Neutrophil) ShouldIncurDamage(ctx context.Context) bool {
	return n.Cell.ShouldIncurDamage(ctx) && n.inNETosis
}

func (n *Neutrophil) Start(ctx context.Context) {
	n.function = n.dna.makeFunction(n)
	go n.function.Run(ctx, n)
	n.Tissue().Attach(n.render)
}

func (n *Neutrophil) BroadcastPosition(ctx context.Context) {
	n.Tissue().BroadcastPosition(ctx, n, n.render)
}

func (n *Neutrophil) Interact(ctx context.Context, c CellActor) {
	antigen := c.PresentAntigen(false)
	if antigen.mollecular_pattern != BACTERIA_MOLECULAR_MOTIF {
		return
	}
	// It's bacteria, time to kill.
	// There are three modes of killing for neutrophils:
	//  1 - Phagocytosis: If the bacteria is covered opsonins (complements
	//      bound to parts of the bacteria), then the neutrophil can consume
	//      it. We won't have complements, so we'll simulate that with by
	//      checking if the bacteria has been around for some time.
	//  2 - Degranulations: Release cytotoxic chemicals to cause damage.
	//  3 - Neutrophil Extracellular Traps: Trap bacteria in a web of innards
	//      made of DNA and toxins to keep them in place and cause damage.
	//
	n.DropCytokine(antigen_present, CYTOKINE_ANTIGEN_PRESENT)
	n.Organ().materialPool.PutLigand(&LigandBlob{
		inflammation: LIGAND_INFLAMMATION_LEUKOCYTE,
	})
	if n.GetCytokineConcentrationAt(antigen_present, c.Position()) >= NEUTROPHIL_NETOSIS_THRESHOLD {
		n.inNETosis = true
	}
	if n.inNETosis {
		n.Trap(c)
		c.IncurDamage(NEUTROPHIL_NET_DAMAGE)
	} else if c.SpawnTime().Add(15 * time.Minute).After(time.Now()) {
		// Enough time has passed that bacteria should be covered in opsonins.
		// Can perform phagocytosis without NET, which is insta kill.
		n.Trap(c)
		c.IncurDamage(MAX_DAMAGE)
	} else {
		n.DropCytokine(cytotoxins, CYTOKINE_CYTOTOXINS)
	}
}

func CopyNeutrophil(base *Neutrophil) *Neutrophil {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &Neutrophil{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
	}
}

type NaturalKiller struct {
	*Leukocyte
}

func (n *NaturalKiller) Start(ctx context.Context) {
	n.function = n.dna.makeFunction(n)
	go n.function.Run(ctx, n)
	n.Tissue().Attach(n.render)
}

func (n *NaturalKiller) BroadcastPosition(ctx context.Context) {
	n.Tissue().BroadcastPosition(ctx, n, n.render)
}

func (n *NaturalKiller) Interact(ctx context.Context, c CellActor) {
	antigen := c.PresentAntigen(false)
	if antigen.mollecular_pattern != BACTERIA_MOLECULAR_MOTIF {
		return
	}
	n.DropCytokine(antigen_present, CYTOKINE_ANTIGEN_PRESENT)
	ligand := n.Organ().materialPool.GetLigand(ctx)
	defer n.Organ().materialPool.PutLigand(ligand)
	ligand.inflammation += LIGAND_INFLAMMATION_LEUKOCYTE
}

func CopyNaturalKiller(base *NaturalKiller) *NaturalKiller {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &NaturalKiller{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
	}
}

type Macrophage struct {
	*Leukocyte
}

func (m *Macrophage) Start(ctx context.Context) {
	m.function = m.dna.makeFunction(m)
	go m.function.Run(ctx, m)
	m.Tissue().Attach(m.render)
}

func (m *Macrophage) BroadcastPosition(ctx context.Context) {
	m.Tissue().BroadcastPosition(ctx, m, m.render)
}

func CopyMacrophage(base *Macrophage) *Macrophage {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &Macrophage{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
	}
}

type TCell struct {
	*Leukocyte
	proteinReceptor Protein
}

func (t *TCell) Start(ctx context.Context) {
	t.function = t.dna.makeFunction(t)
	go t.function.Run(ctx, t)
	t.Tissue().Attach(t.render)
}

func (t *TCell) BroadcastPosition(ctx context.Context) {
	t.Tissue().BroadcastPosition(ctx, t, t.render)
}

func GenerateTCellProteins(dna *DNA) (proteins []Protein) {
	for i := 0; i < 65535; i++ {
		_, isSelf := dna.selfProteins[Protein(i)]
		if !isSelf {
			proteins = append(proteins, Protein(i))
		}
	}
	return
}

func CopyTCell(base *TCell) *TCell {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &TCell{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
		proteinReceptor: base.proteinReceptor,
	}
}

type DendriticCell struct {
	*Leukocyte
	proteinSignatures map[Protein]bool
}

func (d *DendriticCell) Start(ctx context.Context) {
	d.function = d.dna.makeFunction(d)
	go d.function.Run(ctx, d)
	d.Tissue().Attach(d.render)
}

func (d *DendriticCell) BroadcastPosition(ctx context.Context) {
	d.Tissue().BroadcastPosition(ctx, d, d.render)
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

func CopyDendriticCell(base *DendriticCell) *DendriticCell {
	position := image.Point{base.render.position.X, base.render.position.Y}
	positionTracker := ring.New(POSITION_TRACKER_SIZE)
	positionTracker.Value = position
	return &DendriticCell{
		Leukocyte: &Leukocyte{
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
				transportPath: base.transportPath,
				wantPath:      base.wantPath,
				spawnTime:     time.Now(),
			},
			lifeSpan: base.lifeSpan,
		},
		proteinSignatures: base.proteinSignatures,
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
			transportPath: base.transportPath,
			wantPath:      base.wantPath,
			spawnTime:     time.Now(),
		},
		generationTime:     generationTime,
		lastGenerationTime: time.Now(),
	}
}

func (p *ProkaryoticCell) Start(ctx context.Context) {
	tissue := p.Tissue()
	if tissue == nil {
		return
	}
	p.function = p.dna.makeFunction(p)
	go p.function.Run(ctx, p)
	tissue.Attach(p.render)
}

func (p *ProkaryoticCell) DoesWork() bool {
	return false
}

func (p *ProkaryoticCell) CanTransport() bool {
	return true
}

func (p *ProkaryoticCell) BroadcastPosition(ctx context.Context) {
	tissue := p.Tissue()
	if tissue == nil {
		return
	}
	tissue.BroadcastPosition(ctx, p, p.render)
}

func (p *ProkaryoticCell) WillMitosis(context.Context) bool {
	if time.Now().After(p.lastGenerationTime.Add(p.generationTime)) && p.energy >= BACTERIA_ENERGY_MITOSIS_THRESHOLD {
		p.energy = 0
		return true
	}
	return false
}

func (p *ProkaryoticCell) Mitosis(ctx context.Context) bool {
	p.lastGenerationTime = time.Now()
	if p.organ == nil {
		return false
	}
	MakeTransportRequest(p.organ.transportUrl, p.cellType.String(), p.dna, p.cellType, nothing, string(p.render.id), p.transportPath, p.wantPath)
	return true
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

func MakeCellFromType(cellType CellType, workType WorkType, dna *DNA, render *Renderable, transportPath [10]string, wantPath [10]string) (cell CellActor) {
	switch cellType {
	// Bacteria
	case Bacteria:
		fallthrough
	case Bacteroidota:
		cell = CopyProkaryoticCell(&ProkaryoticCell{
			Cell: &Cell{
				cellType:      cellType,
				dna:           dna,
				render:        render,
				transportPath: transportPath,
				wantPath:      wantPath,
				spawnTime:     time.Now(),
			},
		})
	// Leukocytes
	case Lymphoblast:
		fallthrough
	case Myeloblast:
		cell = CopyLeukocyteStemCell(&LeukocyteStemCell{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: LEUKOCYTE_STEM_CELL_LIFE_SPAN,
			},
		})
	case Macrophagocyte:
		cell = CopyMacrophage(&Macrophage{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: MACROPHAGE_LIFE_SPAN,
			},
		})
	case Neutrocytes:
		cell = CopyNeutrophil(&Neutrophil{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: NEUTROPHIL_LIFE_SPAN,
			},
		})
	case LargeGranularLymphocytes:
		cell = CopyNaturalKiller(&NaturalKiller{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: NATURALKILLER_LIFE_SPAN,
			},
		})
	case TLymphocyte:
		cell = CopyTCell(&TCell{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: TCELL_LIFE_SPAN,
			},
		})
	case Dendritic:
		cell = CopyDendriticCell(&DendriticCell{
			Leukocyte: &Leukocyte{
				Cell: &Cell{
					cellType:      cellType,
					dna:           dna,
					workType:      workType,
					render:        render,
					transportPath: transportPath,
					wantPath:      wantPath,
					spawnTime:     time.Now(),
				},
				lifeSpan: DENDRITIC_CELL_LIFE_SPAN,
			},
		})
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
	default:
		cell = CopyEukaryoticCell(&EukaryoticCell{
			Cell: &Cell{
				cellType:      cellType,
				dna:           dna,
				workType:      workType,
				render:        render,
				transportPath: transportPath,
				wantPath:      wantPath,
				spawnTime:     time.Now(),
			},
		})
	}
	return
}
