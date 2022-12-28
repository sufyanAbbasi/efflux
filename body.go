package main

import (
	"context"
)

type Graph struct {
	allNodes map[string]*Node
}

type Body struct {
	*Graph
	bloodNodes  []*Node
	boneNodes   []*Node
	brainNodes  []*Node
	gutNodes    []*Node
	heartNodes  []*Node
	lymphNodes  []*Node
	lungNodes   []*Node
	muscleNodes []*Node
	skinNodes   []*Node
	kidneyNodes []*Node
}

func (b *Body) GenerateCellsAndStart(ctx context.Context) {

	// Generate Eukaryotic Cells
	nodeTypes := [][]*Node{
		b.bloodNodes,
		b.brainNodes,
		b.heartNodes,
		b.lungNodes,
		b.muscleNodes,
		b.skinNodes,
		b.gutNodes,
		b.kidneyNodes,
		b.boneNodes,
	}
	cellTypes := []CellType{
		RedBlood,
		Neuron,
		Cardiomyocyte,
		Pneumocyte,
		Myocyte,
		Keratinocyte,
		Enterocyte,
		Podocyte,
		Hemocytoblast,
	}

	workTypes := []WorkType{
		exchange, // Blood
		think,    // Brain
		pump,     // Heart
		exhale,   // Lungs
		move,     // Muscles
		cover,    // Skin
		digest,   // Gut
		filter,   // Kidney
		nothing,  // BoneMarrow
	}

	counts := []int{
		1, // Blood
		3, // Brain
		1, // Heart
		1, // Lungs
		1, // Muscles
		1, // Skin
		1, // Gut
		1, // Kidney
		1, // BoneMarrow
	}
	humanDNA := MakeDNA(HUMAN_DNA, HUMAN_NAME)
	for i, nodes := range nodeTypes {
		for _, node := range nodes {
			for j := 0; j < counts[i]; j++ {
				MakeTransportRequest(node.transportUrl, HUMAN_NAME, humanDNA, cellTypes[i], workTypes[i], "", [10]string{}, [10]string{})
			}
			if cellTypes[i] == Neuron {
				// Add a Hemocytoblast to the brain, to spawn immune cells.
				MakeTransportRequest(node.transportUrl, HUMAN_NAME, humanDNA, Hemocytoblast, nothing, "", [10]string{}, [10]string{})
			}
		}
	}
	// Generate Prokaryotic Cells
	nodeTypes = [][]*Node{
		b.gutNodes,
	}
	cellTypes = []CellType{
		Bacteroidota,
	}

	counts = []int{
		1, // Gut
	}

	for i, nodes := range nodeTypes {
		for _, node := range nodes {
			cellType := cellTypes[i]
			bacteriaDNA := MakeDNA(BACTERIA_DNA, cellType.String())
			for j := 0; j < counts[i]; j++ {
				MakeTransportRequest(node.transportUrl, cellType.String(), bacteriaDNA, cellTypes[i], nothing, "", [10]string{}, [10]string{})
			}
		}
	}

}

func GenerateBody(ctx context.Context) *Body {
	b := &Body{
		Graph: &Graph{
			allNodes: make(map[string]*Node),
		},
	}
	// Organs
	brain := InitializeNewNode(ctx, b.Graph, "Brain")
	b.brainNodes = append(b.brainNodes, brain)

	heart := InitializeNewNode(ctx, b.Graph, "Heart")
	b.heartNodes = append(b.heartNodes, heart)
	ConnectNodes(ctx, heart, brain, neuronal, neuronal)

	lungLeft := InitializeNewNode(ctx, b.Graph, "Left Lung")
	lungRight := InitializeNewNode(ctx, b.Graph, "Right Lung")
	ConnectNodes(ctx, lungLeft, heart, muscular, muscular)
	ConnectNodes(ctx, lungRight, heart, muscular, muscular)
	b.lungNodes = append(b.lungNodes, lungLeft, lungRight)

	// Kidneys
	kidneyLeft := InitializeNewNode(ctx, b.Graph, "Kidney - Left")
	kidneyRight := InitializeNewNode(ctx, b.Graph, "Kidney - Right")
	b.kidneyNodes = append(b.kidneyNodes, kidneyLeft, kidneyRight)

	// Muscles and Skin

	// Left Arm
	muscleLeftArm := InitializeNewNode(ctx, b.Graph, "Left Arm Muscle")
	skinLeftArm := InitializeNewNode(ctx, b.Graph, "Left Arm Skin")
	ConnectNodes(ctx, muscleLeftArm, skinLeftArm, muscular, muscular)
	ConnectNodes(ctx, muscleLeftArm, brain, neuronal, neuronal)

	// Right Arm
	muscleRightArm := InitializeNewNode(ctx, b.Graph, "Right Arm Muscle")
	skinRightArm := InitializeNewNode(ctx, b.Graph, "Right Arm Skin")
	ConnectNodes(ctx, muscleRightArm, skinRightArm, muscular, muscular)
	ConnectNodes(ctx, muscleRightArm, brain, neuronal, neuronal)

	// Left Leg
	muscleLeftLeg := InitializeNewNode(ctx, b.Graph, "Left Leg Muscle")
	skinLeftLeg := InitializeNewNode(ctx, b.Graph, "Left Leg Skin")
	ConnectNodes(ctx, muscleLeftLeg, skinLeftLeg, muscular, muscular)
	ConnectNodes(ctx, muscleLeftLeg, brain, neuronal, neuronal)

	// Right Leg
	muscleRightLeg := InitializeNewNode(ctx, b.Graph, "Right Leg Muscle")
	skinRightLeg := InitializeNewNode(ctx, b.Graph, "Right Leg Skin")
	ConnectNodes(ctx, muscleRightLeg, skinRightLeg, muscular, muscular)
	ConnectNodes(ctx, muscleRightLeg, brain, neuronal, neuronal)

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
	ConnectNodes(ctx, bloodBrain, brain, blood_brain_barrier, blood_brain_barrier)
	ConnectNodes(ctx, bloodBrain, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodBrain, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodBrain, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodBrain, kidneyRight, muscular, cardiovascular)
	bloodHeart := InitializeNewNode(ctx, b.Graph, "Blood - Heart")
	ConnectNodes(ctx, bloodHeart, heart, muscular, cardiovascular)
	ConnectNodes(ctx, bloodHeart, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodHeart, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodHeart, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodHeart, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodBrain, bloodHeart, cardiovascular, cardiovascular)
	bloodLung := InitializeNewNode(ctx, b.Graph, "Blood - Lung")
	ConnectNodes(ctx, bloodLung, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLung, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLung, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLung, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLung, bloodHeart, cardiovascular, cardiovascular)
	bloodTorso := InitializeNewNode(ctx, b.Graph, "Blood - Torso")
	ConnectNodes(ctx, bloodTorso, bloodLung, cardiovascular, cardiovascular)
	ConnectNodes(ctx, bloodTorso, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodTorso, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodTorso, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodTorso, kidneyRight, muscular, cardiovascular)
	bloodLeftArm := InitializeNewNode(ctx, b.Graph, "Blood - Left Arm")
	ConnectNodes(ctx, bloodLeftArm, muscleLeftArm, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftArm, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftArm, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftArm, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftArm, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftArm, bloodTorso, cardiovascular, cardiovascular)
	bloodRightArm := InitializeNewNode(ctx, b.Graph, "Blood - Right Arm")
	ConnectNodes(ctx, bloodRightArm, muscleRightArm, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightArm, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightArm, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightArm, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightArm, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightArm, bloodTorso, cardiovascular, cardiovascular)
	bloodLeftLeg := InitializeNewNode(ctx, b.Graph, "Blood - Left Leg")
	ConnectNodes(ctx, bloodLeftLeg, muscleLeftLeg, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftLeg, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftLeg, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftLeg, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftLeg, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodLeftLeg, bloodTorso, cardiovascular, cardiovascular)
	bloodRightLeg := InitializeNewNode(ctx, b.Graph, "Blood - Right Leg")
	ConnectNodes(ctx, bloodRightLeg, muscleRightLeg, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightLeg, lungLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightLeg, lungRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightLeg, kidneyLeft, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightLeg, kidneyRight, muscular, cardiovascular)
	ConnectNodes(ctx, bloodRightLeg, bloodTorso, cardiovascular, cardiovascular)
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
	ConnectNodes(ctx, lymphHeart, bloodHeart, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphHeart, heart, muscular, lymphatic)
	lymphLung := InitializeNewNode(ctx, b.Graph, "Lymph Node - Lung")
	ConnectNodes(ctx, lymphLung, bloodLung, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphLung, lymphHeart, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphLung, lungLeft, muscular, lymphatic)
	ConnectNodes(ctx, lymphLung, lungRight, muscular, lymphatic)
	lymphTorso := InitializeNewNode(ctx, b.Graph, "Lymph Node - Torso")
	ConnectNodes(ctx, lymphTorso, bloodTorso, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphTorso, lymphLung, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphTorso, kidneyLeft, muscular, lymphatic)
	ConnectNodes(ctx, lymphTorso, kidneyRight, muscular, lymphatic)
	lymphLeftArm := InitializeNewNode(ctx, b.Graph, "Lymph Node - Left Arm")
	ConnectNodes(ctx, lymphLeftArm, bloodLeftArm, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphLeftArm, lymphTorso, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphLeftArm, muscleLeftArm, muscular, lymphatic)
	lymphRightArm := InitializeNewNode(ctx, b.Graph, "Lymph Node - Right Arm")
	ConnectNodes(ctx, lymphRightArm, bloodRightArm, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphRightArm, lymphTorso, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphRightArm, muscleRightArm, muscular, lymphatic)
	lymphLeftLeg := InitializeNewNode(ctx, b.Graph, "Lymph Node - Left Leg")
	ConnectNodes(ctx, lymphLeftLeg, bloodLeftLeg, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphLeftLeg, lymphTorso, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphLeftLeg, muscleLeftLeg, cardiovascular, lymphatic)
	lymphRightLeg := InitializeNewNode(ctx, b.Graph, "Lymph Node - Right Leg")
	ConnectNodes(ctx, lymphRightLeg, bloodRightLeg, cardiovascular, lymphatic)
	ConnectNodes(ctx, lymphRightLeg, lymphTorso, lymphatic, lymphatic)
	ConnectNodes(ctx, lymphRightLeg, muscleRightLeg, cardiovascular, lymphatic)
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
	ConnectNodes(ctx, boneLeftArm, bloodLeftArm, cardiovascular, skeletal)
	boneRightArm := InitializeNewNode(ctx, b.Graph, "Bone - Right Arm")
	ConnectNodes(ctx, boneRightArm, bloodRightArm, cardiovascular, skeletal)
	boneLeftLeg := InitializeNewNode(ctx, b.Graph, "Bone - Left Leg")
	ConnectNodes(ctx, boneLeftLeg, bloodLeftLeg, cardiovascular, skeletal)
	boneRightLeg := InitializeNewNode(ctx, b.Graph, "Bone - Right Leg")
	ConnectNodes(ctx, boneRightLeg, bloodRightLeg, cardiovascular, skeletal)
	b.boneNodes = append(b.boneNodes,
		boneLeftArm,
		boneRightArm,
		boneLeftLeg,
		boneRightLeg,
	)

	// Gut
	gut := InitializeNewNode(ctx, b.Graph, "Gut")
	ConnectNodes(ctx, gut, bloodTorso, cardiovascular, muscular)
	ConnectNodes(ctx, gut, lymphTorso, lymphatic, muscular)
	ConnectNodes(ctx, gut, muscleLeftArm, muscular, muscular)
	ConnectNodes(ctx, gut, muscleRightArm, muscular, muscular)
	ConnectNodes(ctx, gut, muscleLeftLeg, muscular, muscular)
	ConnectNodes(ctx, gut, muscleRightLeg, muscular, muscular)
	b.gutNodes = append(b.gutNodes, gut)

	b.GenerateCellsAndStart(ctx)
	return b
}
