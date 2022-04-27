package main

import (
	"testing"
)

func TestDendriticCellFindTCells(t *testing.T) {
	bacteriaDNA := MakeDNA(BACTERIA_DNA, "E. Coli")
	virusRNA := MakeDNA(VIRUS_RNA, "COVID-19")
	human2DNA := MakeDNA(HUMAN_DNA, "Human 2")
	humanDNA := MakeDNA(HUMAN_DNA, "Human 1")

	dendriticCell := MakeDendriticCell(humanDNA)

	dendriticCell.Collect(MakeProkaryoticCell(bacteriaDNA, Bacterial))
	dendriticCell.Collect(MakeVirus(virusRNA, Viral))
	dendriticCell.Collect(MakeEukaryoticStemCell(human2DNA, Pneumocyte, 0))

	tCells := GenerateTCells(humanDNA)

	for _, t := range tCells {
		dendriticCell.FoundMatch(t)
	}

	for signature, found := range dendriticCell.proteinSignatures {
		if !found {
			t.Errorf("Did not find %v", signature)
		}
	}
}
