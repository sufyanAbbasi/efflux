package main

import (
	"context"
)

type EdgeType int // Determines how Nodes are connected.

const (
	cellular    EdgeType = iota // Cell to cell connection
	bloodVessel                 // Connected via blood vessels
	lymphVessel                 // Connected via lymph vessels
	neurons                     // Connected via neurons.
)

const (
	status WorkType = iota
	cover           // Called on skin cells by muscle cells. Will randomly fail, i.e. cuts.
	inhale          // Called on lung cells by blood cells.
	exhale          // Called on blood cells by other cells.
	pump            // Called on to heart cells to pump, by brain cels.
	move            // Called on muscle cells by brain cells.
	think           // Called on brain cells to perform a computation, by muscle cells.
)

type Graph struct {
	allNodes map[string]*Node
}

type Body struct {
	*Graph
	dna         *DNA
	bloodNodes  []*Node
	boneNodes   []*Node
	brainNodes  []*Node
	heartNodes  []*Node
	lymphNodes  []*Node
	lungNodes   []*Node
	muscleNodes []*Node
	skinNodes   []*Node
}

func (b *Body) GenerateCellsAndStart(ctx context.Context) {
	nodeTypes := [][]*Node{
		b.bloodNodes,
		b.brainNodes,
		b.heartNodes,
		b.lungNodes,
		b.muscleNodes,
		b.skinNodes,
	}
	cellTypes := []CellType{
		RedBlood,
		Neuron,
		Cardiomyocyte,
		Pneumocyte,
		Myocyte,
		Keratinocyte,
	}

	workTypes := []WorkType{
		inhale, // Blood
		think,  // Brain
		pump,   // Heart
		exhale, // Lungs
		move,   // Muscles
		cover,  // Skin
	}

	counts := []int{
		100, // Blood
		50,  // Brain
		10,  // Heart
		10,  // Lungs
		10,  // Muscles
		10,  // Skin
	}
	for i, nodes := range nodeTypes {
		for _, node := range nodes {
			for j := 0; j < counts[i]; j++ {
				cell := MakeEukaryoticStemCell(b.dna, cellTypes[i], workTypes[i])
				cell.parent = node
				go cell.Start(ctx)
			}
		}
	}
}

func GenerateBody(ctx context.Context) *Body {
	b := &Body{
		Graph: &Graph{
			allNodes: make(map[string]*Node),
		},
		dna: MakeDNA(HUMAN_DNA, HUMAN_NAME),
	}
	// Organs
	brain := InitializeNewNode(ctx, b.Graph, "Brain")
	b.brainNodes = append(b.brainNodes, brain)

	heart := InitializeNewNode(ctx, b.Graph, "Heart")
	b.heartNodes = append(b.heartNodes, heart)
	ConnectNodes(ctx, heart, brain)

	lungLeft := InitializeNewNode(ctx, b.Graph, "Left Lung")
	b.lungNodes = append(b.lungNodes, lungLeft)

	lungRight := InitializeNewNode(ctx, b.Graph, "Right Lung")
	b.lungNodes = append(b.lungNodes, lungRight)

	// Muscles and Skin

	// Left Arm
	muscleLeftArm := InitializeNewNode(ctx, b.Graph, "Left Arm Muscle")
	skinLeftArm := InitializeNewNode(ctx, b.Graph, "Left Arm Skin")
	ConnectNodes(ctx, muscleLeftArm, skinLeftArm)
	ConnectNodes(ctx, muscleLeftArm, brain)

	// Right Arm
	muscleRightArm := InitializeNewNode(ctx, b.Graph, "Right Arm Muscle")
	skinRightArm := InitializeNewNode(ctx, b.Graph, "Right Arm Skin")
	ConnectNodes(ctx, muscleRightArm, skinRightArm)
	ConnectNodes(ctx, muscleRightArm, brain)

	// Left Leg
	muscleLeftLeg := InitializeNewNode(ctx, b.Graph, "Left Leg Muscle")
	skinLeftLeg := InitializeNewNode(ctx, b.Graph, "Left Leg Skin")
	ConnectNodes(ctx, muscleLeftLeg, skinLeftLeg)
	ConnectNodes(ctx, muscleLeftLeg, brain)

	// Right Leg
	muscleRightLeg := InitializeNewNode(ctx, b.Graph, "Right Leg Muscle")
	skinRightLeg := InitializeNewNode(ctx, b.Graph, "Right Leg Skin")
	ConnectNodes(ctx, muscleRightLeg, skinRightLeg)
	ConnectNodes(ctx, muscleRightLeg, brain)

	b.muscleNodes = append(b.muscleNodes,
		muscleLeftArm,
		muscleRightArm,
		muscleLeftLeg,
		muscleRightLeg,
	)
	b.skinNodes = append(b.skinNodes,
		skinLeftArm,
		skinRightArm,
		skinLeftLeg,
		skinRightLeg,
	)

	// Blood

	bloodBrain := InitializeNewNode(ctx, b.Graph, "Blood - Brain")
	ConnectNodes(ctx, bloodBrain, brain)
	bloodHeart := InitializeNewNode(ctx, b.Graph, "Blood - Heart")
	ConnectNodes(ctx, bloodHeart, heart)
	ConnectNodes(ctx, bloodBrain, bloodHeart)
	bloodLung := InitializeNewNode(ctx, b.Graph, "Blood - Lung")
	ConnectNodes(ctx, bloodLung, lungLeft)
	ConnectNodes(ctx, bloodLung, lungRight)
	ConnectNodes(ctx, bloodLung, bloodHeart)
	bloodTorso := InitializeNewNode(ctx, b.Graph, "Blood - Torso")
	ConnectNodes(ctx, bloodTorso, bloodLung)
	ConnectNodes(ctx, bloodTorso, lungLeft)
	ConnectNodes(ctx, bloodTorso, lungRight)
	bloodLeftArm := InitializeNewNode(ctx, b.Graph, "Blood - Left Arm")
	ConnectNodes(ctx, bloodLeftArm, muscleLeftArm)
	ConnectNodes(ctx, bloodLeftArm, bloodTorso)
	bloodRightArm := InitializeNewNode(ctx, b.Graph, "Blood - Right Arm")
	ConnectNodes(ctx, bloodRightArm, muscleRightArm)
	ConnectNodes(ctx, bloodRightArm, bloodTorso)
	bloodLeftLeg := InitializeNewNode(ctx, b.Graph, "Blood - Left Leg")
	ConnectNodes(ctx, bloodLeftLeg, muscleLeftLeg)
	ConnectNodes(ctx, bloodLeftLeg, bloodTorso)
	bloodRightLeg := InitializeNewNode(ctx, b.Graph, "Blood - Right Leg")
	ConnectNodes(ctx, bloodRightLeg, muscleRightLeg)
	ConnectNodes(ctx, bloodRightLeg, bloodTorso)
	b.bloodNodes = append(b.bloodNodes,
		bloodBrain,
		bloodHeart,
		bloodLung,
		bloodTorso,
		bloodLeftArm,
		bloodRightArm,
		bloodLeftLeg,
		bloodRightLeg,
	)

	// Lymph Nodes

	lymphHeart := InitializeNewNode(ctx, b.Graph, "Lymph Node - Heart")
	ConnectNodes(ctx, lymphHeart, bloodHeart)
	ConnectNodes(ctx, lymphHeart, heart)
	lymphLung := InitializeNewNode(ctx, b.Graph, "Lymph Node - Lung")
	ConnectNodes(ctx, lymphLung, bloodLung)
	ConnectNodes(ctx, lymphLung, lymphHeart)
	ConnectNodes(ctx, lymphLung, lungLeft)
	ConnectNodes(ctx, lymphLung, lungRight)
	lymphTorso := InitializeNewNode(ctx, b.Graph, "Lymph Node - Torso")
	ConnectNodes(ctx, lymphTorso, bloodTorso)
	ConnectNodes(ctx, lymphTorso, lymphLung)
	lymphLeftArm := InitializeNewNode(ctx, b.Graph, "Lymph Node - Left Arm")
	ConnectNodes(ctx, lymphLeftArm, bloodLeftArm)
	ConnectNodes(ctx, lymphLeftArm, lymphTorso)
	ConnectNodes(ctx, lymphLeftArm, muscleLeftArm)
	lymphRightArm := InitializeNewNode(ctx, b.Graph, "Lymph Node - Right Arm")
	ConnectNodes(ctx, lymphRightArm, bloodRightArm)
	ConnectNodes(ctx, lymphRightArm, lymphTorso)
	ConnectNodes(ctx, lymphRightArm, muscleRightArm)
	lymphLeftLeg := InitializeNewNode(ctx, b.Graph, "Lymph Node - Left Leg")
	ConnectNodes(ctx, lymphLeftLeg, bloodLeftLeg)
	ConnectNodes(ctx, lymphLeftLeg, lymphTorso)
	ConnectNodes(ctx, lymphLeftLeg, muscleLeftLeg)
	lymphRightLeg := InitializeNewNode(ctx, b.Graph, "Lymph Node - Right Leg")
	ConnectNodes(ctx, lymphRightLeg, bloodRightLeg)
	ConnectNodes(ctx, lymphRightLeg, lymphTorso)
	ConnectNodes(ctx, lymphRightLeg, muscleRightLeg)
	b.lymphNodes = append(b.lymphNodes,
		lymphHeart,
		lymphLung,
		lymphTorso,
		lymphLeftArm,
		lymphRightArm,
		lymphLeftLeg,
		lymphRightLeg,
	)

	// Bone Marrow
	boneLeftArm := InitializeNewNode(ctx, b.Graph, "Bone - Left Arm")
	ConnectNodes(ctx, boneLeftArm, bloodLeftArm)
	boneRightArm := InitializeNewNode(ctx, b.Graph, "Bone - Right Arm")
	ConnectNodes(ctx, boneRightArm, bloodRightArm)
	boneLeftLeg := InitializeNewNode(ctx, b.Graph, "Bone - Left Leg")
	ConnectNodes(ctx, boneLeftLeg, bloodLeftLeg)
	boneRightLeg := InitializeNewNode(ctx, b.Graph, "Bone - Right Leg")
	ConnectNodes(ctx, boneRightLeg, bloodRightLeg)
	b.boneNodes = append(b.boneNodes,
		boneLeftArm,
		boneRightArm,
		boneLeftLeg,
		boneRightLeg,
	)

	b.GenerateCellsAndStart(ctx)
	return b
}
