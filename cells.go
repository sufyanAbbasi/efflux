package main

import "fmt"

type Cell struct {
	cellType string
	dna      *DNA
	mhc_i    MHC_I
	antigen  Antigen
	workType WorkType
	parent   *Node
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

type AntigenPresenting interface {
	PresentAntigen() Antigen
	DNA() *DNA
}

func (c *Cell) PresentAntigen() Antigen {
	if c.antigen == nil {
		c.antigen = c.dna.GenerateAntigen()
	}
	return c.antigen
}

func (c *Cell) DNA() *DNA {
	return c.dna
}

type HumanCell struct {
	*Cell
}

func makeHumanCell(dna *DNA) *HumanCell {
	return &HumanCell{
		Cell: &Cell{
			cellType: "HumanCell",
			dna:      dna,
			mhc_i:    dna.MHC_I(),
		},
	}
}

type Bacteria struct {
	Cell
}

func makeBacteria(dna *DNA) *Bacteria {
	return &Bacteria{
		Cell: Cell{
			cellType: "Bacteria",
			dna:      dna,
			mhc_i:    dna.MHC_I(),
		},
	}
}

type Virus struct {
	Cell
}

func makeVirus(rna *DNA) *Virus {
	return &Virus{
		Cell: Cell{
			cellType: "Virus",
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

func makeImmuneCell(dna *DNA) *ImmuneCell {
	return &ImmuneCell{
		Cell: Cell{
			cellType: "ImmuneCell",
			dna:      dna,
			mhc_i:    dna.MHC_I(),
		},
	}
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
	antigenSignature AntigenSignature
}

func makeTCell(dna *DNA, antigenSignature AntigenSignature) *TCell {
	return &TCell{
		ImmuneCell: ImmuneCell{
			Cell: Cell{
				cellType: "TCell",
				dna:      dna,
				mhc_i:    dna.MHC_I(),
			},
		},
		antigenSignature: antigenSignature,
	}
}

func generateTCells(dna *DNA) (tCells []*TCell) {
	for i := 0; i < 65535; i++ {
		tCells = append(tCells, makeTCell(dna, AntigenSignature(i)))
	}
	return
}

type DendriticCell struct {
	ImmuneCell
	antigenSignatures map[AntigenSignature]bool
}

func (d *DendriticCell) Collect(t AntigenPresenting) {
	d.antigenSignatures[t.DNA().antigenSignature] = false
}

func (d *DendriticCell) FoundMatch(t *TCell) bool {
	_, found := d.antigenSignatures[t.antigenSignature]
	if found {
		d.antigenSignatures[t.antigenSignature] = found
	}
	return found
}

func makeDendriticCell(dna *DNA) *DendriticCell {
	return &DendriticCell{
		ImmuneCell: ImmuneCell{
			Cell: Cell{
				cellType: "DendriticCell",
				dna:      dna,
				mhc_i:    dna.MHC_I(),
			},
		},
		antigenSignatures: make(map[AntigenSignature]bool),
	}
}
