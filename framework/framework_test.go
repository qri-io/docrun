package framework

import (
	"io"
	"io/ioutil"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"

	"testing"
)

// The actual DocRun processor, given each node as they are parsed.
var runner DocRunner

// renderHook is called by markdown parser for each ast node.
func renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	// Only process nodes when they open up.
	if entering {
		runner.AddNode(node)
	}
	// Tell the parser to continue walking the ast normally.
	return ast.GoToNext, false
}

func TestDocrunner(t *testing.T) {
	opts := html.RendererOptions{
		Flags:          html.CommonFlags,
		RenderNodeHook: renderHook,
	}
	renderer := html.NewRenderer(opts)
	md, err := ioutil.ReadFile("testdata/doc.md")
	if err != nil {
		panic(err)
	}
	runner.Init()
	_ = markdown.ToHTML([]byte(md), nil, renderer)
	if runner.HasError() {
		runner.ShowErrors()
		t.Errorf("Docrunner encountered errors")
	}
	res := runner.GetResults()
	if res.CountTotal != 1 {
		t.Errorf("Expected 1 total test, got %d", res.CountTotal)
	}
	if res.CountSuccess != 1 {
		t.Errorf("Expected 1 successful test, got %d", res.CountSuccess)
	}
	if res.CountTrivial != 1 {
		t.Errorf("Expected 1 trivial test, got %d", res.CountTrivial)
	}
	if res.CountMissing != 0 {
		t.Errorf("Expected 0 missing tests, got %d", res.CountMissing)
	}
}
