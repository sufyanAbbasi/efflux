package main

import (
	"testing"
)

func TestCellVerification(t *testing.T) {
	bacteriaDNA := MakeDNA(BACTERIA_DNA, "E. Coli")
	bacteria := &ProkaryoticCell{
		Cell: &Cell{
			cellType: Bacteria,
			dna:      bacteriaDNA,
			mhc_i:    bacteriaDNA.MHC_I(),
		},
	}

	virusRNA := MakeDNA(VIRUS_RNA, "COVID-19")
	virus := MakeVirus(virusRNA, nil, Pneumocyte)

	humanDNA := MakeDNA(HUMAN_DNA, "Human 1")
	humanCell := &EukaryoticCell{
		Cell: &Cell{
			cellType: Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}

	infectedHumanCell := &EukaryoticCell{
		Cell: &Cell{
			cellType: Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}
	virus.InfectCell(infectedHumanCell)

	tCell := &Leukocyte{
		Cell: &Cell{
			cellType: Pneumocyte,
			dna:      humanDNA,
			mhc_i:    humanDNA.MHC_I(),
		},
	}

	human2DNA := MakeDNA(HUMAN_DNA, "Human 2")
	human2Cell := &EukaryoticCell{
		Cell: &Cell{
			cellType: Pneumocyte,
			dna:      human2DNA,
			mhc_i:    human2DNA.MHC_I(),
		},
	}

	cases := []struct {
		name      string
		got, want bool
	}{
		{"tCell", tCell.CheckAntigen(tCell), true},
		{"humanCell", tCell.CheckAntigen(humanCell), true},
		{"human2Cell", tCell.CheckAntigen(human2Cell), false},
		{"bacteria", tCell.CheckAntigen(bacteria), false},
		{"virus", tCell.CheckAntigen(infectedHumanCell), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%v == %v, want %v", c.name, c.got, c.want)
		}
	}

	virus.InfectCell(humanCell)
	cases = []struct {
		name      string
		got, want bool
	}{
		{"humanCell", tCell.CheckAntigen(humanCell), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%v == %v, want %v", c.name, c.got, c.want)
		}
	}

}
