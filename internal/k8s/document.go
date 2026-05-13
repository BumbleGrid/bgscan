package k8s

import (
	"fmt"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
)

const defaultBGSpecVersion = "0.1"

func placeholderFloor(level int) floor.Content {
	return floor.Content{
		Floor:       level,
		Label:       fmt.Sprintf("Floor %d", level),
		Description: fmt.Sprintf("Placeholder floor %d (no scan data).", level),
		Nodes:       []node.Wrapper{},
		Edges:       []edge.Wrapper{},
		Meta: &floor.BlockMeta{
			Status:           "placeholder",
			Notes:            "Reserved for higher-level graph views.",
			ExtractorVersion: "",
			ExtractedAt:      "",
		},
	}
}

func NewBGSpecDocument(floor0 floor.Content) graph.BGSpecDocument {
	return graph.BGSpecDocument{
		BGSpec: defaultBGSpecVersion,
		Document: graph.DocumentMeta{
			Title:     "Kubernetes scan",
			Company:   "",
			UpdatedAt: "1970-01-01T00:00:00Z",
		},
		Floors: []floor.Content{
			floor0,
			placeholderFloor(1),
			placeholderFloor(2),
			placeholderFloor(3),
		},
	}
}
