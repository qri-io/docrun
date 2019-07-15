// Package framework is the main implementation of docrun. It accumulates data about test fixtures
// along with source code extracted from markdown files. Then it runs that source code and makes
// sure the results match what is expected.
package framework

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/fill"
	"gopkg.in/yaml.v2"
)

// DocRunner maintains state to process nodes, run examples, and collect results
type DocRunner struct {
	Errs        []error
	Fixture     *DocrunFixture
	Source      *DocrunSource
	CaseError   bool
	Results     RunResults
	Starlark    *StarlarkRunner
	CommandLine *CommandLineRunner
}

// RunResults collects results from a run of docrun
type RunResults struct {
	CountTotal   int
	CountSuccess int
	CountTrivial int
	CountMissing int
}

// AddSuccess counts up a successfully ran case
func (r *RunResults) AddSuccess(caseNum int, nonTrivial bool) {
	r.CountSuccess++
	if !nonTrivial {
		r.CountTrivial++
	}
}

// AddMissing counts that a case doesn't have a fixture
func (r *RunResults) AddMissing() {
	r.CountMissing++
}

// Empty returns whether there were no tests run at all
func (r *RunResults) Empty() bool {
	return r.CountTotal == 0
}

// Init assigns initial state to the DocRunner
func (f *DocRunner) Init() {
	f.Errs = []error{}
	f.Fixture = nil
	f.Source = nil
	f.CaseError = false
	f.Results = RunResults{}
	f.Starlark = NewStarlarkRunner()
	f.CommandLine = NewCommandLineRunner()
}

// HandleNode is given each parsed ast node, and collects information about tests to run
func (f *DocRunner) HandleNode(node ast.Node) (*DocrunFixture, *DocrunSource, error) {
	// A fixture begins with an HTML comment block containing metadata about a test to run.
	if _, ok := node.(*ast.HTMLBlock); ok {
		leaf := node.AsLeaf()
		if leaf != nil {
			text := strings.Trim(string(leaf.Literal), " \n")
			if strings.HasPrefix(text, "<!--") && strings.HasSuffix(text, "-->") {
				text = text[4 : len(text)-3]
				text = strings.TrimSpace(text)

				// Only run over comment blocks that start with the string "docrun".
				if !strings.HasPrefix(text, "docrun") {
					return nil, nil, nil
				}

				var fields map[string]interface{}
				err := yaml.Unmarshal([]byte(text), &fields)
				if err != nil {
					return nil, nil, err
				}

				fixture := DocrunFixture{}
				err = fill.Struct(fields, &fixture)
				if err != nil {
					return nil, nil, err
				}
				return &fixture, nil, nil
			}
		}
	}

	// A code block is treated as source code that can be run, as long as it has valid metadata.
	if cb, ok := node.(*ast.CodeBlock); ok {
		leaf := node.AsLeaf()
		if leaf != nil {
			return nil, &DocrunSource{string(leaf.Literal), string(cb.Info)}, nil
		}
	}

	return nil, nil, nil
}

// AddNode collects a node, either a fixture represented by a HTML comment block, or source code.
func (f *DocRunner) AddNode(node ast.Node) {
	fixture, source, err := f.HandleNode(node)
	if err != nil {
		f.AddError(err)
		f.CaseError = true
		return
	}
	if f.Fixture == nil && fixture != nil {
		// Hold onto fixture until the source code is also parsed.
		f.Fixture = fixture
		return
	}
	if f.Source == nil && source != nil {
		// Once fixture and source are available, run the test case.
		f.Source = source
		f.RunFixture()
		f.ClearState()
	}
}

// ClearState finishes a fixture run by clearing the related state.
func (f *DocRunner) ClearState() {
	f.Fixture = nil
	f.Source = nil
	f.CaseError = false
}

// RunFixture runs a fixture by combining metadata and the source code.
func (f *DocRunner) RunFixture() {
	f.Results.CountTotal++
	if f.Fixture == nil {
		// If this fixture case already encountered an an error, don't throw another.
		if f.CaseError {
			return
		}
		// Source code blocks should all be immediately preceded by a fixture node. It is an error
		// to have source code without a fixture node. An easy to silence this is to add:
		// <!--
		// docrun:
		//   pass: true
		// -->
		f.AddError(fmt.Errorf("source code block %d is not preceded by a docrun fixture",
			f.Results.CountTotal))
		f.Results.AddMissing()
		return
	}
	if f.Fixture.Docrun.Pass {
		// A trivially passing test.
		f.Results.AddSuccess(f.Results.CountTotal, false)
		return
	}

	lang := ""
	// If source code has a language tag, use that for the source language.
	if f.Source.Lang != "" {
		lang = f.Source.Lang
	}
	// Otherwise, if top-level of fixture has a language field, use that.
	// TODO(dlong): Should it be an error to have both set?
	if lang == "" && f.Fixture.Docrun.Lang != "" {
		lang = f.Fixture.Docrun.Lang
	}
	if lang == "" {
		f.AddError(fmt.Errorf("source code block %d has no language", f.Results.CountTotal))
		return
	}

	// If there's a filltype, parse the text using that type to make sure it is valid.
	filltype := f.Fixture.Docrun.Filltype
	if filltype != "" {
		f.DispatchFilltype(filltype, f.Source.Code)
		f.HandleSave(f.Fixture.Docrun.Save, f.Source.Code)
		return
	}
	// If there's a test substructure, dispatch it.
	test := f.Fixture.Docrun.Test
	if test != nil {
		f.DispatchTestCase(test, lang, f.Source.Code)
		f.HandleSave(f.Fixture.Docrun.Save, f.Source.Code)
		return
	}
	// If there's a command, dispatch it.
	cmd := f.Fixture.Docrun.Command
	if cmd != nil {
		f.DispatchCommandCase(cmd, lang, f.Source.Code)
		f.HandleSave(f.Fixture.Docrun.Save, f.Source.Code)
		return
	}
	f.HandleSave(f.Fixture.Docrun.Save, f.Source.Code)
	f.Results.AddSuccess(f.Results.CountTotal, false)
}

// HandleSave saves the source code to a file, for future tests and commands.
func (f *DocRunner) HandleSave(save *saveDetails, sourceCode string) {
	if save == nil {
		return
	}
	if save.Append {
		// TODO(dlong): Implement appending this source to the file
	} else {
		// TODO(dlong): Implement overwriting the file
	}
}

// HasError returns whether there were any errors
func (f *DocRunner) HasError() bool {
	return len(f.Errs) > 0
}

// ShowErrors displays errors to stdout
func (f *DocRunner) ShowErrors() {
	for _, err := range f.Errs {
		fmt.Printf("Error: %s\n\n", err)
	}
}

// AddError adds an error
func (f *DocRunner) AddError(err error) {
	// TODO(dlong): Clean up how this interacts with ShowErrors, which prefixes "Error: "
	f.Errs = append(f.Errs, fmt.Errorf("case %d: %s", f.Results.CountTotal, err))
}

// DisplayResults displays results from running the test cases.
func (f *DocRunner) DisplayResults() {
	if f.Results.CountTrivial == 0 {
		fmt.Printf("PASS: %d tests\n", f.Results.CountSuccess)
	} else {
		fmt.Printf("PASS: %d tests (%d trivial)\n", f.Results.CountSuccess, f.Results.CountTrivial)
	}
	failNum := f.Results.CountTotal - f.Results.CountSuccess
	if f.Results.CountMissing == 0 {
		fmt.Printf("FAIL: %d\n", failNum)
	} else {
		fmt.Printf("FAIL: %d (%d missing)\n", failNum, f.Results.CountMissing)
	}
}

// GetResults returns the results from a run of docrun
func (f *DocRunner) GetResults() RunResults {
	return f.Results
}

// DispatchFilltype dispatches a filltype operation.
func (f *DocRunner) DispatchFilltype(filltype, source string) {
	var err error
	var fields map[string]interface{}
	switch filltype {
	case "json":
		err = json.Unmarshal([]byte(source), &fields)
	case "dataset.Dataset":
		// TODO(dlong): Support datasets in json format
		err = yaml.Unmarshal([]byte(source), &fields)
		if err == nil {
			ds := dataset.Dataset{}
			err = fill.Struct(fields, &ds)
		}
	default:
		err = fmt.Errorf("unknown filltype %s", filltype)
	}
	if err != nil {
		f.AddError(err)
	} else {
		f.Results.AddSuccess(f.Results.CountTotal, true)
	}
}

// DispatchTestCase dispatches a test case.
func (f *DocRunner) DispatchTestCase(test *testDetails, lang, source string) {
	var err error
	switch lang {
	case "python":
		err = f.Starlark.Run(test, source)
	default:
		err = fmt.Errorf("unknown code language %s", lang)
	}
	if err != nil {
		f.AddError(err)
	} else {
		f.Results.AddSuccess(f.Results.CountTotal, true)
	}
}

// DispatchCommandCase dispatches a command.
// TODO(dlong): Implementation is only a stub currently.
func (f *DocRunner) DispatchCommandCase(cmd *commandDetails, lang, source string) {
	var err error
	switch lang {
	case "shell":
		err = f.CommandLine.Run(cmd, source)
	default:
		err = fmt.Errorf("unknown code language %s", lang)
	}
	if err != nil {
		f.AddError(err)
	} else {
		f.Results.AddSuccess(f.Results.CountTotal, true)
	}
}
