package main

import (
	"context"
)

var currentPort = 8000

type Graph struct {
	allNodes map[string]*Node
}

type Body struct {
	*Graph
	bloodNodes  []*Node
	boneNodes   []*Node
	brainNodes  []*Node
	heartNodes  []*Node
	lymphNodes  []*Node
	lungNodes   []*Node
	muscleNodes []*Node
	skinNodes   []*Node
}

func GenerateBody(ctx context.Context) *Body {
	b := &Body{
		Graph: &Graph{
			allNodes: make(map[string]*Node),
		},
	}
	// Organs
	brain := InitializeNewNode(ctx, b.Graph, "brain")
	b.brainNodes = append(b.brainNodes, brain)

	heart := InitializeNewNode(ctx, b.Graph, "heart")
	b.heartNodes = append(b.heartNodes, heart)

	lungLeft := InitializeNewNode(ctx, b.Graph, "lungLeft")
	b.lungNodes = append(b.lungNodes, lungLeft)

	lungRight := InitializeNewNode(ctx, b.Graph, "lungRight")
	b.lungNodes = append(b.lungNodes, lungRight)

	// Muscles and Skin

	// Left Arm
	muscleLeftArm := InitializeNewNode(ctx, b.Graph, "muscleLeftArm")
	skinLeftArm := InitializeNewNode(ctx, b.Graph, "skinLeftArm")
	ConnectNodes(ctx, muscleLeftArm, skinLeftArm)

	// Right Arm
	muscleRightArm := InitializeNewNode(ctx, b.Graph, "muscleRightArm")
	skinRightArm := InitializeNewNode(ctx, b.Graph, "skinRightArm")
	ConnectNodes(ctx, muscleRightArm, skinRightArm)

	// Left Leg
	muscleLeftLeg := InitializeNewNode(ctx, b.Graph, "muscleLeftLeg")
	skinLeftLeg := InitializeNewNode(ctx, b.Graph, "skinLeftLeg")
	ConnectNodes(ctx, muscleLeftLeg, skinLeftLeg)

	// Right Leg
	muscleRightLeg := InitializeNewNode(ctx, b.Graph, "muscleRightLeg")
	skinRightLeg := InitializeNewNode(ctx, b.Graph, "skinRightLeg")
	ConnectNodes(ctx, muscleRightLeg, skinRightLeg)

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

	bloodBrain := InitializeNewNode(ctx, b.Graph, "bloodBrain")
	ConnectNodes(ctx, bloodBrain, brain)
	bloodHeart := InitializeNewNode(ctx, b.Graph, "bloodHeart")
	ConnectNodes(ctx, bloodHeart, heart)
	ConnectNodes(ctx, bloodBrain, bloodHeart)
	bloodLung := InitializeNewNode(ctx, b.Graph, "bloodLung")
	ConnectNodes(ctx, bloodLung, lungLeft)
	ConnectNodes(ctx, bloodLung, lungRight)
	ConnectNodes(ctx, bloodLung, bloodHeart)
	bloodTorso := InitializeNewNode(ctx, b.Graph, "bloodTorso")
	ConnectNodes(ctx, bloodTorso, bloodLung)
	ConnectNodes(ctx, bloodTorso, lungLeft)
	ConnectNodes(ctx, bloodTorso, lungRight)
	bloodLeftArm := InitializeNewNode(ctx, b.Graph, "bloodLeftArm")
	ConnectNodes(ctx, bloodLeftArm, muscleLeftArm)
	ConnectNodes(ctx, bloodLeftArm, bloodTorso)
	bloodRightArm := InitializeNewNode(ctx, b.Graph, "bloodRightArm")
	ConnectNodes(ctx, bloodRightArm, muscleRightArm)
	ConnectNodes(ctx, bloodRightArm, bloodTorso)
	bloodLeftLeg := InitializeNewNode(ctx, b.Graph, "bloodLeftLeg")
	ConnectNodes(ctx, bloodLeftLeg, muscleLeftLeg)
	ConnectNodes(ctx, bloodLeftLeg, bloodTorso)
	bloodRightLeg := InitializeNewNode(ctx, b.Graph, "bloodRightLeg")
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

	lymphHeart := InitializeNewNode(ctx, b.Graph, "lymphHeart")
	ConnectNodes(ctx, lymphHeart, bloodHeart)
	lymphLung := InitializeNewNode(ctx, b.Graph, "lymphLung")
	ConnectNodes(ctx, lymphLung, bloodLung)
	ConnectNodes(ctx, lymphLung, lymphHeart)
	lymphTorso := InitializeNewNode(ctx, b.Graph, "lymphTorso")
	ConnectNodes(ctx, lymphTorso, bloodTorso)
	ConnectNodes(ctx, lymphTorso, lymphLung)
	lymphLeftArm := InitializeNewNode(ctx, b.Graph, "lymphLeftArm")
	ConnectNodes(ctx, lymphLeftArm, bloodLeftArm)
	ConnectNodes(ctx, lymphLeftArm, lymphTorso)
	lymphRightArm := InitializeNewNode(ctx, b.Graph, "lymphRightArm")
	ConnectNodes(ctx, lymphRightArm, bloodRightArm)
	ConnectNodes(ctx, lymphRightArm, lymphTorso)
	lymphLeftLeg := InitializeNewNode(ctx, b.Graph, "lymphLeftLeg")
	ConnectNodes(ctx, lymphLeftLeg, bloodLeftLeg)
	ConnectNodes(ctx, lymphLeftLeg, lymphTorso)
	lymphRightLeg := InitializeNewNode(ctx, b.Graph, "lymphRightLeg")
	ConnectNodes(ctx, lymphRightLeg, bloodRightLeg)
	ConnectNodes(ctx, lymphRightLeg, lymphTorso)
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
	boneLeftArm := InitializeNewNode(ctx, b.Graph, "boneLeftArm")
	ConnectNodes(ctx, boneLeftArm, bloodLeftArm)
	boneRightArm := InitializeNewNode(ctx, b.Graph, "boneRightArm")
	ConnectNodes(ctx, boneRightArm, bloodRightArm)
	boneLeftLeg := InitializeNewNode(ctx, b.Graph, "boneLeftLeg")
	ConnectNodes(ctx, boneLeftLeg, bloodLeftLeg)
	boneRightLeg := InitializeNewNode(ctx, b.Graph, "boneRightLeg")
	ConnectNodes(ctx, boneRightLeg, bloodRightLeg)
	b.boneNodes = append(b.boneNodes,
		boneLeftArm,
		boneRightArm,
		boneLeftLeg,
		boneRightLeg,
	)

	return b
}
