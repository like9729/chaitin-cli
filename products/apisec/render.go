package apisec

import (
	"encoding/json"
	"fmt"
	"io"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

type Renderer struct {
	format Format
	out    io.Writer
}

func NewRenderer(format Format, out io.Writer) Renderer {
	return Renderer{format: format, out: out}
}

func (r Renderer) Render(value any) error {
	var data []byte
	var err error
	if r.format == FormatJSON {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("render response: %w", err)
	}
	_, err = fmt.Fprintln(r.out, string(data))
	return err
}
