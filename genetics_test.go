package main

import (
	"testing"
)

func TestCellVerification(t *testing.T) {
	bacteriaDNA := makeDNA(BACTERIA_DNA, "E. Coli")
	bacteria := makeBacteria(bacteriaDNA)

	virusRNA := makeDNA(VIRUS_RNA, "COVID-19")
	virus := makeVirus(virusRNA)

	humanDNA := makeDNA(HUMAN_DNA, "Human 1")
	humanCell := makeHumanCell(humanDNA)
	tCell := makeTCell(humanDNA, 0)

	human2DNA := makeDNA(HUMAN_DNA, "Human 2")
	human2Cell := makeHumanCell(human2DNA)

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
