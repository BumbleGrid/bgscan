package output

import "io"

type Writer struct{}

func NewWriter(out io.Writer) *Writer {
	return &Writer{}
}
