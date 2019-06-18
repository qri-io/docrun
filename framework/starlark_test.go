package framework

import (
	"strings"
	"testing"
)

func TestStarlarkRun(t *testing.T) {
	runner := NewStarlarkRunner()

	details := testDetails{
		Call:   "transform(ds, ctx)",
		Actual: "ds.get_body()",
		Expect: []string{"1", "2", "3"},
	}

	// A successful test case.
	sourceCode := `
def transform(ds, ctx):
  ds.set_body(["1","2","3"])`
	err := runner.Run(&details, sourceCode)
	if err != nil {
		t.Fatal(err)
	}

	// Failure due to having the wrong result.
	sourceCode = `
def transform(ds, ctx):
  ds.set_body(["1","2","3","4"])`
	err = runner.Run(&details, sourceCode)
	if err == nil {
		t.Fatalf("Expect test to fail, did not receive error")
	}
	expectText := `actual: [1 2 3 4]`
	if !strings.Contains(err.Error(), expectText) {
		t.Errorf("Error does not contain expected \"%s\"\nerror: \"%s\"", expectText, err.Error())
	}
	expectText = `expect: [1 2 3`
	if !strings.Contains(err.Error(), expectText) {
		t.Errorf("Error does not contain expected \"%s\"\nerror: \"%s\"", expectText, err.Error())
	}

	// Failure due to having incorrect parameters for transform function.
	sourceCode = `
def transform(ds):
  ds.set_body(["1","2","3"])`
	err = runner.Run(&details, sourceCode)
	if err == nil {
		t.Fatalf("Expect test to fail, did not receive error")
	}
	expectText = `function transform accepts 1 positional argument`
	if !strings.Contains(err.Error(), expectText) {
		t.Errorf("Error does not contain expected \"%s\"\nerror: \"%s\"", expectText, err.Error())
	}

	// Failure due to calling `set_body` incorrectly.
	sourceCode = `
def transform(ds, ctx):
  ds.set_body("text")`
	err = runner.Run(&details, sourceCode)
	if err == nil {
		t.Fatalf("Expect test to fail, did not receive error")
	}
	expectText = `expected body data to be iterable`
	if !strings.Contains(err.Error(), expectText) {
		t.Errorf("Error does not contain expected \"%s\"\nerror: \"%s\"", expectText, err.Error())
	}

	// Failure due to having parameters in the wrong order.
	sourceCode = `
def transform(ctx, ds):
  ds.set_body(["1","2","3"])`
	err = runner.Run(&details, sourceCode)
	if err == nil {
		t.Fatalf("Expect test to fail, did not receive error")
	}
	expectText = `"context" struct has no .set_body attribute`
	if !strings.Contains(err.Error(), expectText) {
		t.Errorf("Error does not contain expected \"%s\"\nerror: \"%s\"", expectText, err.Error())
	}
}
