package main

import (
	"testing"
)

func TestCellVerification(t *testing.T) {
	bacteriaDNA := MakeDNA(BACTERIA_DNA, "E. Coli")
	bacteria := MakeProkaryoticCell(bacteriaDNA, Bacterial)

	virusRNA := MakeDNA(VIRUS_RNA, "COVID-19")
	virus := MakeVirus(virusRNA, Viral)

	humanDNA := MakeDNA(HUMAN_DNA, "Human 1")
	humanCell := MakeEukaryoticStemCell(humanDNA, Pneumocyte, 0)
	tCell := MakeTCell(humanDNA, 0)

	human2DNA := MakeDNA(HUMAN_DNA, "Human 2")
	human2Cell := MakeEukaryoticStemCell(human2DNA, Pneumocyte, 0)

	cases := []struct {
		name      string
		got, want bool
	}{
		{"tCell", tCell.CheckAntigen(tCell), true},
		{"humanCell", tCell.CheckAntigen(humanCell), true},
		{"human2Cell", tCell.CheckAntigen(human2Cell), false},
		{"bacteria", tCell.CheckAntigen(bacteria), false},
		{"virus", tCell.CheckAntigen(virus), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%v == %v, want %v", c.name, c.got, c.want)
		}
	}

	virus.InfectCell(humanCell.Cell)
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
