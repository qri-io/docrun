package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/docrun/framework"
)

// The actual DocRun processor, given each node as they are parsed.
var runner framework.DocRunner

// renderHook is called by markdown parser for each ast node.
func renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	// Only process nodes when they open up.
	if entering {
		runner.AddNode(node)
	}
	// Tell the parser to continue walking the ast normally.
	return ast.GoToNext, false
}

func setLogLevel(logLevel int) {
	// Assign log level to the logger.
	if logLevel == 1 {
		golog.SetLogLevel("docrun", "info")
	} else if logLevel == 2 {
		golog.SetLogLevel("docrun", "debug")
	}
}

func createRunResults(path string) {
	// This library converts markdown to html, with a hook for parsed nodes. We ignore the html
	// output, and only care about the ast nodes while parsing is happening.
	opts := html.RendererOptions{
		Flags:          html.CommonFlags,
		RenderNodeHook: renderHook,
	}
	renderer := html.NewRenderer(opts)
	md, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Printf("File not found: \"%s\"\n", path)
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}
	// Parse markdown to collect and run test cases.
	_ = markdown.ToHTML([]byte(md), nil, renderer)
}

func docAnalyze(path string) {
	runner.Init()
	createRunResults(path)
	if runner.HasError() {
		runner.ShowErrors()
	}
	runner.DisplayResults()
}

func docGetResults(path string) framework.RunResults {
	runner.Init()
	createRunResults(path)
	return runner.GetResults()
}
