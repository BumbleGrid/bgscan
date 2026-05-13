// Package output serializes the assembled BGSpec Floor 0 graph (nodes +
// edges) to JSON and writes it to a destination (file path or stdout).
//
// The emitted document must validate against bgspec/bgspec.schema.json;
// keep field ordering and required keys aligned with that schema.
package output

import "io"

// Writer renders a BGSpec graph to an io.Writer as JSON.
type Writer struct {
	// TODO: hold formatting options (pretty-print indent, trailing
	// newline, schema version stamp).
}

// NewWriter returns a Writer that emits to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{}
}

// Write serializes the graph (nodes, edges, floor metadata) to the
// underlying writer. The exact graph type will come from bgbase/graph
// once wired in.
//
// TODO: Write(graph graph.Graph) error
