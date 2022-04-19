package main

import (
	"testing"
)

func TestDendriticCellFindTCells(t *testing.T) {
	bacteriaDNA := makeDNA(BACTERIA_DNA, "E. Coli")
	virusRNA := makeDNA(VIRUS_RNA, "COVID-19")
	human2DNA := makeDNA(HUMAN_DNA, "Human 2")
	humanDNA := makeDNA(HUMAN_DNA, "Human 1")

	dendriticCell := makeDendriticCell(humanDNA)

	dendriticCell.Collect(makeBacteria(bacteriaDNA))
	dendriticCell.Collect(makeVirus(virusRNA))
	dendriticCell.Collect(makeHumanCell(human2DNA))

	tCells := generateTCells(humanDNA)

	for _, t := range tCells {
		dendriticCell.FoundMatch(t)
	}

	for signature, found := range dendriticCell.antigenSignatures {
		if !found {
			t.Errorf("Did not find %v", signature)
		}
	}
}
