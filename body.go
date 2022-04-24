package main

import (
	"context"
	"fmt"
)

var currentPort = 8000

func GetNextAddresses() (string, string, string, string) {
	currentPort++
	return ORIGIN,
		fmt.Sprintf(":%v", currentPort),
		fmt.Sprintf(URL_TEMPLATE, currentPort),
		fmt.Sprintf(WEBSOCKET_URL_TEMPLATE, currentPort)
}

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

func InitializeNewNode(ctx context.Context, graph *Graph) *Node {
	origin, port, url, websocketUrl := GetNextAddresses()
	node := &Node{
		origin:       origin,
		port:         port,
		websocketUrl: websocketUrl,
		managers:     make(map[WorkType]*Manager),
	}
	graph.allNodes[url] = node
	node.Start(ctx)
	return node
}

func ConnectNodes(ctx context.Context, node1, node2 *Node) {
	node1.Connect(ctx, node2.origin, node1.websocketUrl)
	node2.Connect(ctx, node1.origin, node1.websocketUrl)
}

func GenerateBody(ctx context.Context) *Body {
	b := &Body{
		Graph: &Graph{
			allNodes: make(map[string]*Node),
		},
	}
	// Organs
	brain := InitializeNewNode(ctx, b.Graph)
	b.brainNodes = append(b.brainNodes, brain)

	heart := InitializeNewNode(ctx, b.Graph)
	b.heartNodes = append(b.heartNodes, heart)

	lungLeft := InitializeNewNode(ctx, b.Graph)
	b.lungNodes = append(b.lungNodes, lungLeft)

	lungRight := InitializeNewNode(ctx, b.Graph)
	b.lungNodes = append(b.lungNodes, lungRight)

	// Muscles and Skin

	// Left Arm
	muscleLeftArm := InitializeNewNode(ctx, b.Graph)
	skinLeftArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, muscleLeftArm, skinLeftArm)

	// Right Arm
	muscleRightArm := InitializeNewNode(ctx, b.Graph)
	skinRightArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, muscleRightArm, skinRightArm)

	// Left Leg
	muscleLeftLeg := InitializeNewNode(ctx, b.Graph)
	skinLeftLeg := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, muscleLeftLeg, skinLeftLeg)

	// Right Leg
	muscleRightLeg := InitializeNewNode(ctx, b.Graph)
	skinRightLeg := InitializeNewNode(ctx, b.Graph)
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

	bloodBrain := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodBrain, brain)
	bloodHeart := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodHeart, heart)
	ConnectNodes(ctx, bloodBrain, bloodHeart)
	bloodLung := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodLung, lungLeft)
	ConnectNodes(ctx, bloodLung, lungRight)
	ConnectNodes(ctx, bloodLung, bloodHeart)
	bloodTorso := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodTorso, bloodLung)
	ConnectNodes(ctx, bloodTorso, lungLeft)
	ConnectNodes(ctx, bloodTorso, lungRight)
	bloodLeftArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodLeftArm, muscleLeftArm)
	ConnectNodes(ctx, bloodLeftArm, bloodTorso)
	bloodRightArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodRightArm, muscleRightArm)
	ConnectNodes(ctx, bloodRightArm, bloodTorso)
	bloodLeftLeg := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, bloodLeftLeg, muscleLeftLeg)
	ConnectNodes(ctx, bloodLeftLeg, bloodTorso)
	bloodRightLeg := InitializeNewNode(ctx, b.Graph)
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

	lymphHeart := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphHeart, bloodHeart)
	lymphLung := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphLung, bloodLung)
	ConnectNodes(ctx, lymphLung, lymphHeart)
	lymphTorso := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphTorso, bloodTorso)
	ConnectNodes(ctx, lymphTorso, lymphLung)
	lymphLeftArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphLeftArm, bloodLeftArm)
	ConnectNodes(ctx, lymphLeftArm, lymphTorso)
	lymphRightArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphRightArm, bloodRightArm)
	ConnectNodes(ctx, lymphRightArm, lymphTorso)
	lymphLeftLeg := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, lymphLeftLeg, bloodLeftLeg)
	ConnectNodes(ctx, lymphLeftLeg, lymphTorso)
	lymphRightLeg := InitializeNewNode(ctx, b.Graph)
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
	boneLeftArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, boneLeftArm, bloodLeftArm)
	boneRightArm := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, boneRightArm, bloodRightArm)
	boneLeftLeg := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, boneLeftLeg, bloodLeftLeg)
	boneRightLeg := InitializeNewNode(ctx, b.Graph)
	ConnectNodes(ctx, boneRightLeg, bloodRightLeg)
	b.boneNodes = append(b.boneNodes,
		boneLeftArm,
		boneRightArm,
		boneLeftLeg,
		boneRightLeg,
	)

	return b
}
