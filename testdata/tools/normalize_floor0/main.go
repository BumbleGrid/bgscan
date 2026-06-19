package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/scanner/k8s"
	"github.com/BumbleGrid/bgbase/validate"
)

func main() {
	inputPath := flag.String("input", "", "Path to bgscan floor-0 JSON output")
	goldenPath := flag.String("golden", "", "Path to expected floor0.golden.json")
	check := flag.Bool("check", false, "Normalize, validate, and compare to golden")
	updateGolden := flag.Bool("update-golden", false, "Normalize, validate, and write golden file")
	flag.Parse()

	if *inputPath == "" {
		exitErr(fmt.Errorf("--input is required"))
	}
	if *check && *updateGolden {
		exitErr(fmt.Errorf("--check and --update-golden are mutually exclusive"))
	}

	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		exitErr(fmt.Errorf("read input: %w", err))
	}

	content, err := parseFloorContent(raw)
	if err != nil {
		exitErr(err)
	}

	normalized := normalizeFloorContent(content)
	if err := validateFloorContent(normalized); err != nil {
		exitErr(fmt.Errorf("schema validation failed: %w", err))
	}

	payload, err := graph.MarshalFloorContentJSON(normalized)
	if err != nil {
		exitErr(fmt.Errorf("marshal normalized output: %w", err))
	}

	switch {
	case *updateGolden:
		if *goldenPath == "" {
			exitErr(fmt.Errorf("--golden is required with --update-golden"))
		}
		if err := os.WriteFile(*goldenPath, payload, 0o644); err != nil {
			exitErr(fmt.Errorf("write golden: %w", err))
		}
		fmt.Fprintf(os.Stderr, "normalize_floor0: updated golden %s\n", *goldenPath)
	case *check:
		if *goldenPath == "" {
			exitErr(fmt.Errorf("--golden is required with --check"))
		}
		goldenRaw, err := os.ReadFile(*goldenPath)
		if err != nil {
			exitErr(fmt.Errorf("read golden: %w", err))
		}
		goldenContent, err := parseFloorContent(goldenRaw)
		if err != nil {
			exitErr(fmt.Errorf("parse golden: %w", err))
		}
		goldenPayload, err := graph.MarshalFloorContentJSON(normalizeFloorContent(goldenContent))
		if err != nil {
			exitErr(fmt.Errorf("marshal golden: %w", err))
		}
		if !bytes.Equal(payload, goldenPayload) {
			exitErr(fmt.Errorf("floor-0 output differs from golden %s", *goldenPath))
		}
		fmt.Fprintf(os.Stderr, "normalize_floor0: ok — matches golden %s\n", *goldenPath)
	default:
		if _, err := os.Stdout.Write(payload); err != nil {
			exitErr(fmt.Errorf("write stdout: %w", err))
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			exitErr(fmt.Errorf("write stdout: %w", err))
		}
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "normalize_floor0:", err)
	os.Exit(1)
}

func parseFloorContent(raw []byte) (floor.Content, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return floor.Content{}, fmt.Errorf("decode input: %w", err)
	}
	if _, wholeDocument := probe["bgspec"]; wholeDocument {
		var doc graph.BGSpecDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			return floor.Content{}, fmt.Errorf("decode whole document: %w", err)
		}
		if len(doc.Floors) == 0 {
			return floor.Content{}, fmt.Errorf("whole document has no floors")
		}
		return doc.Floors[0], nil
	}
	var content floor.Content
	if err := json.Unmarshal(raw, &content); err != nil {
		return floor.Content{}, fmt.Errorf("decode floor content: %w", err)
	}
	return content, nil
}

func normalizeFloorContent(content floor.Content) floor.Content {
	out := content
	if out.Meta != nil {
		meta := *out.Meta
		meta.ExtractedAt = ""
		meta.ExtractorVersion = ""
		out.Meta = &meta
	}

	out.Nodes = make([]node.Wrapper, len(content.Nodes))
	for idx := range content.Nodes {
		wrapper := content.Nodes[idx]
		data := wrapper.Data
		if data.Meta != nil {
			meta := *data.Meta
			meta.ExtractedAt = ""
			meta.ExtractorVersion = ""
			data.Meta = &meta
		}
		if data.K8s != nil {
			k8sMeta := *data.K8s
			k8sMeta.UID = ""
			data.K8s = &k8sMeta
		}
		out.Nodes[idx] = node.Wrapper{Data: data}
	}

	out.Edges = make([]edge.Wrapper, len(content.Edges))
	for idx := range content.Edges {
		wrapper := content.Edges[idx]
		data := wrapper.Data
		data.Meta.ExtractedAt = ""
		data.Meta.ExtractorVersion = ""
		out.Edges[idx] = edge.Wrapper{Data: data}
	}

	out.Nodes, out.Edges = dropVolatileCronJobNodes(out.Nodes, out.Edges)

	sort.Slice(out.Nodes, func(left, right int) bool {
		return out.Nodes[left].Data.ID < out.Nodes[right].Data.ID
	})
	sort.Slice(out.Edges, func(left, right int) bool {
		return out.Edges[left].Data.ID < out.Edges[right].Data.ID
	})

	return out
}

func dropVolatileCronJobNodes(nodes []node.Wrapper, edges []edge.Wrapper) ([]node.Wrapper, []edge.Wrapper) {
	dropIDs := make(map[string]struct{})
	filteredNodes := make([]node.Wrapper, 0, len(nodes))
	for idx := range nodes {
		nodeID := nodes[idx].Data.ID
		if isVolatileCronJobNodeID(nodeID) {
			dropIDs[nodeID] = struct{}{}
			continue
		}
		filteredNodes = append(filteredNodes, nodes[idx])
	}
	if len(dropIDs) == 0 {
		return nodes, edges
	}
	filteredEdges := make([]edge.Wrapper, 0, len(edges))
	for idx := range edges {
		edgeData := edges[idx].Data
		if _, drop := dropIDs[edgeData.Source]; drop {
			continue
		}
		if _, drop := dropIDs[edgeData.Target]; drop {
			continue
		}
		filteredEdges = append(filteredEdges, edges[idx])
	}
	return filteredNodes, filteredEdges
}

func isVolatileCronJobNodeID(nodeID string) bool {
	const jobsSegment = "/jobs/jobs/"
	segmentIdx := strings.LastIndex(nodeID, jobsSegment)
	if segmentIdx < 0 {
		return false
	}
	jobName := nodeID[segmentIdx+len(jobsSegment):]
	dashIdx := strings.LastIndex(jobName, "-")
	if dashIdx < 0 {
		return false
	}
	suffix := jobName[dashIdx+1:]
	if len(suffix) < 8 {
		return false
	}
	for charIdx := range suffix {
		ch := suffix[charIdx]
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validateFloorContent(content floor.Content) error {
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		return fmt.Errorf("create schema validator: %w", err)
	}
	specDoc := k8s.NewBGSpecDocument(content)
	return validator.Validate(&specDoc)
}
