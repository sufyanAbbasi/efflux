package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"sync"
)

type CellType int

const (
	Bacterial CellType = iota
	Viral
	RedBlood
	Neuron
	Cardiomyocyte // Heart Cell
	Pneumocyte    // Pulmonary Cell
	Myocyte       // Muscle Cell
	Keratinocyte  // Skin Cell
	TLymphocyte   // T Cell
	Dendritic     // Dendritic Cells
)

func (c CellType) String() string {
	switch c {
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
}

func (c *Cell) SetParent(node *Node) {
	c.parent = node
}

func (c *Cell) GetWorkType() WorkType {
	return c.workType
}

func (c *Cell) String() string {
	return fmt.Sprintf("%v (%v)", c.cellType, c.dna.name)
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
		if resource.o2 >= 6 {
			resource.o2 = 0
		} else {
			resource.o2 -= 6
		}
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

func (c *Cell) SampleProteins() []Protein {
	// TODO: return a random sample of the internal protein state.
	return []Protein{Protein(rand.Uint32()), Protein(rand.Uint32()), Protein(rand.Uint32())}
}

func (c *Cell) PresentAntigen() *Antigen {
	if c.antigen == nil {
		c.antigen = c.dna.GenerateAntigen(c.SampleProteins())
	}
	return c.antigen
}

func (c *Cell) ResetResourceNeed() {
	switch c.cellType {
	case Keratinocyte:
		fallthrough
	case RedBlood:
		c.resourceNeed = &ResourceBlob{
			o2: 0,
		}
	case Neuron:
		fallthrough
	case Pneumocyte:
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
			resource.Consume(c.resourceNeed)
			c.parent.materialPool.PutResource(resource)
		}
	}
	c.ResetResourceNeed()
	return true
}

func (c *Cell) ProduceWaste() {
	waste := c.parent.materialPool.GetWaste()
	waste.co2 += 6
	c.parent.materialPool.PutWaste(waste)
}

func (c *Cell) DNA() *DNA {
	return c.dna
}

type EukaryoticCell struct {
	*Cell
	telomereLength int
	hasTelomerase  bool
	function       *StateDiagram
}

func (e *EukaryoticCell) Start(ctx context.Context) {
	e.function = e.dna.makeFunction(e)
	go e.function.Run(ctx, e)
}

func (e *EukaryoticCell) Mitosis() *EukaryoticCell {
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
	return cell
}

func (e *EukaryoticCell) Apoptosis() {
	if e.parent != nil {
		e.parent.RemoveWorker(e)
	}
	// TODO: make sure this gets garbage collected.
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
}

func (p *ProkaryoticCell) Mitosis() *ProkaryoticCell {
	return MakeProkaryoticCell(p.dna, p.cellType)
}

func MakeProkaryoticCell(dna *DNA, cellType CellType) *ProkaryoticCell {
	return &ProkaryoticCell{
		Cell: Cell{
			cellType: cellType,
			dna:      dna,
			mhc_i:    dna.MHC_I(),
		},
	}
}

type Virus struct {
	Cell
}

func MakeVirus(rna *DNA, cellType CellType) *Virus {
	return &Virus{
		Cell: Cell{
			cellType: cellType,
			dna:      rna,
			mhc_i:    rna.MHC_I(),
		},
	}
}

func (v *Virus) InfectCell(c *Cell) {
	c.dna = v.dna
	c.antigen = nil
	c.PresentAntigen()
}

type ImmuneCell struct {
	Cell
}

type AntigenPresenting interface {
	PresentAntigen() *Antigen
	DNA() *DNA
}

func (i *ImmuneCell) CheckAntigen(c AntigenPresenting) bool {
	if c.PresentAntigen() == nil ||
		!i.dna.Verify(i.mhc_i, c.PresentAntigen()) {
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
	selfProteins := dna.GenerateSelfProteins()
	for i := 0; i < 65535; i++ {
		_, isSelf := selfProteins[Protein(i)]
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
	for _, p := range t.PresentAntigen().proteins {
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
