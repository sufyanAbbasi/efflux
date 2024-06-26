package main

import (
	"testing"
)

func TestCellVerification(t *testing.T) {
	bacteriaDNA := MakeDNA(BACTERIA_DNA, "E. Coli")
	bacteria := &ProkaryoticCell{
		Cell: &Cell{
			cellType: CellType_Bacteria,
			dna:      bacteriaDNA,
			mhc_i:    bacteriaDNA.MHC_I(),
		},
	}

	virusRNA := MakeDNA(VIRUS_RNA, "COVID-19")
	virus := &Virus{
		dna:            virusRNA,
		targetCellType: CellType_Pneumocyte,
	}

	humanDNA := MakeDNA(HUMAN_DNA, "Human 1")
	humanCell := &EukaryoticCell{
		Cell: &Cell{
			cellType: CellType_Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}

	infectedHumanCell := &EukaryoticCell{
		Cell: &Cell{
			cellType: CellType_Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}
	virus.Infect(infectedHumanCell)

	tCell := &Leukocyte{
		Cell: &Cell{
			cellType: CellType_Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}

	human2DNA := MakeDNA(HUMAN_DNA, "Human 2")
	human2Cell := &EukaryoticCell{
		Cell: &Cell{
			cellType: CellType_Pneumocyte,
			dna:      human2DNA,
			mhc_i:    human2DNA.MHC_I(),
		},
	}

	cases := []struct {
		name      string
		got, want bool
	}{
		{"tCell", tCell.VerifySelf(tCell.PresentAntigen()), true},
		{"humanCell", tCell.VerifySelf(humanCell.PresentAntigen()), true},
		{"human2Cell", tCell.VerifySelf(human2Cell.PresentAntigen()), false},
		{"bacteria", tCell.VerifySelf(bacteria.PresentAntigen()), false},
		{"virus", tCell.VerifySelf(infectedHumanCell.PresentAntigen()), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%v == %v, want %v", c.name, c.got, c.want)
		}
	}

	virus.Infect(humanCell)
	cases = []struct {
		name      string
		got, want bool
	}{
		{"humanCell", tCell.VerifySelf(humanCell.PresentAntigen()), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%v == %v, want %v", c.name, c.got, c.want)
		}
	}

}
