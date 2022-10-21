package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
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

func MakeTransportRequest(
	transportUrl string,
	name string,
	dna *DNA,
	cellType CellType,
	workType WorkType,
) {
	dnaBase, err := dna.Serialize()
	if err != nil {
		log.Fatal("Transport: ", err)
	}
	dnaType := 0
	for i, d := range DNATypeMap {
		if d == dna.dnaType {
			dnaType = i
		}
	}
	jsonData, err := json.Marshal(TransportRequest{
		Name:     name,
		Base:     dnaBase,
		DNAType:  dnaType,
		CellType: cellType,
		WorkType: workType,
	})
	if err != nil {
		log.Fatal("Transport: ", err)
	}
	request, err := http.NewRequest("POST", transportUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Transport: ", err)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatal("Transport: ", err)
	}
	defer response.Body.Close()
	// body, _ := ioutil.ReadAll(response.Body)
	// fmt.Println("Body:", string(body))
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
	}

	counts := []int{
		1, // Blood
		1, // Brain
		1, // Heart
		1, // Lungs
		1, // Muscles
		1, // Skin
		1, // Gut
		1, // Kidney
	}
	humanDNA := MakeDNA(HUMAN_DNA, HUMAN_NAME)
	for i, nodes := range nodeTypes {
		for _, node := range nodes {
			for j := 0; j < counts[i]; j++ {
				MakeTransportRequest(node.transportUrl, HUMAN_NAME, humanDNA, cellTypes[i], workTypes[i])
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
				MakeTransportRequest(node.transportUrl, cellType.String(), bacteriaDNA, cellTypes[i], 0)
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
	ConnectNodes(ctx, heart, brain)

	lungLeft := InitializeNewNode(ctx, b.Graph, "Left Lung")
	lungRight := InitializeNewNode(ctx, b.Graph, "Right Lung")
	b.lungNodes = append(b.lungNodes, lungLeft, lungRight)

	// Kidneys
	kidneyLeft := InitializeNewNode(ctx, b.Graph, "Kidney - Left")
	kidneyRight := InitializeNewNode(ctx, b.Graph, "Kidney - Right")
	b.kidneyNodes = append(b.kidneyNodes, kidneyLeft, kidneyRight)

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
	ConnectNodes(ctx, bloodBrain, lungLeft)
	ConnectNodes(ctx, bloodBrain, lungRight)
	ConnectNodes(ctx, bloodBrain, kidneyLeft)
	ConnectNodes(ctx, bloodBrain, kidneyRight)
	bloodHeart := InitializeNewNode(ctx, b.Graph, "Blood - Heart")
	ConnectNodes(ctx, bloodHeart, heart)
	ConnectNodes(ctx, bloodHeart, lungLeft)
	ConnectNodes(ctx, bloodHeart, lungRight)
	ConnectNodes(ctx, bloodHeart, kidneyLeft)
	ConnectNodes(ctx, bloodHeart, kidneyRight)
	ConnectNodes(ctx, bloodBrain, bloodHeart)
	bloodLung := InitializeNewNode(ctx, b.Graph, "Blood - Lung")
	ConnectNodes(ctx, bloodLung, lungLeft)
	ConnectNodes(ctx, bloodLung, lungRight)
	ConnectNodes(ctx, bloodLung, kidneyLeft)
	ConnectNodes(ctx, bloodLung, kidneyRight)
	ConnectNodes(ctx, bloodLung, bloodHeart)
	bloodTorso := InitializeNewNode(ctx, b.Graph, "Blood - Torso")
	ConnectNodes(ctx, bloodTorso, bloodLung)
	ConnectNodes(ctx, bloodTorso, lungLeft)
	ConnectNodes(ctx, bloodTorso, lungRight)
	ConnectNodes(ctx, bloodTorso, kidneyLeft)
	ConnectNodes(ctx, bloodTorso, kidneyRight)
	bloodLeftArm := InitializeNewNode(ctx, b.Graph, "Blood - Left Arm")
	ConnectNodes(ctx, bloodLeftArm, muscleLeftArm)
	ConnectNodes(ctx, bloodLeftArm, lungLeft)
	ConnectNodes(ctx, bloodLeftArm, lungRight)
	ConnectNodes(ctx, bloodLeftArm, kidneyLeft)
	ConnectNodes(ctx, bloodLeftArm, kidneyRight)
	ConnectNodes(ctx, bloodLeftArm, bloodTorso)
	bloodRightArm := InitializeNewNode(ctx, b.Graph, "Blood - Right Arm")
	ConnectNodes(ctx, bloodRightArm, muscleRightArm)
	ConnectNodes(ctx, bloodRightArm, lungLeft)
	ConnectNodes(ctx, bloodRightArm, lungRight)
	ConnectNodes(ctx, bloodRightArm, kidneyLeft)
	ConnectNodes(ctx, bloodRightArm, kidneyRight)
	ConnectNodes(ctx, bloodRightArm, bloodTorso)
	bloodLeftLeg := InitializeNewNode(ctx, b.Graph, "Blood - Left Leg")
	ConnectNodes(ctx, bloodLeftLeg, muscleLeftLeg)
	ConnectNodes(ctx, bloodLeftLeg, lungLeft)
	ConnectNodes(ctx, bloodLeftLeg, lungRight)
	ConnectNodes(ctx, bloodLeftLeg, kidneyLeft)
	ConnectNodes(ctx, bloodLeftLeg, kidneyRight)
	ConnectNodes(ctx, bloodLeftLeg, bloodTorso)
	bloodRightLeg := InitializeNewNode(ctx, b.Graph, "Blood - Right Leg")
	ConnectNodes(ctx, bloodRightLeg, muscleRightLeg)
	ConnectNodes(ctx, bloodRightLeg, lungLeft)
	ConnectNodes(ctx, bloodRightLeg, lungRight)
	ConnectNodes(ctx, bloodRightLeg, kidneyLeft)
	ConnectNodes(ctx, bloodRightLeg, kidneyRight)
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
	ConnectNodes(ctx, lymphTorso, kidneyLeft)
	ConnectNodes(ctx, lymphTorso, kidneyRight)
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

	// Gut
	gut := InitializeNewNode(ctx, b.Graph, "Gut")
	ConnectNodes(ctx, gut, bloodTorso)
	ConnectNodes(ctx, gut, lymphTorso)
	ConnectNodes(ctx, gut, muscleLeftArm)
	ConnectNodes(ctx, gut, muscleRightArm)
	ConnectNodes(ctx, gut, muscleLeftLeg)
	ConnectNodes(ctx, gut, muscleRightLeg)
	b.gutNodes = append(b.gutNodes, gut)

	b.GenerateCellsAndStart(ctx)
	return b
}
