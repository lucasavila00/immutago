package strings

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Text string
type Alias = string

type Catalog struct {
	supervise.Service[string]
	text string
}

func (*Catalog) Run(context.Context) {}
func (c *Catalog) Text() string      { return c.text }
func (*Catalog) Named() Text         { return "text" }
func (*Catalog) Alias() [0]Alias     { return [0]Alias{} }

// read remains ordinary code because it is not exported.
func (c *Catalog) read(text string) string { return c.text + text[:1] }
