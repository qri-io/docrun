package framework

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	stards "github.com/qri-io/qri/startf/ds"
	starutil "github.com/qri-io/starlib/util"
	"github.com/qri-io/startf/context"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var log = golog.Logger("docrun")

// StarlarkRunner collects methods for running starlark code
type StarlarkRunner struct {
}

// NewStarlarkRunner returns a new StarlarkRunner
func NewStarlarkRunner() *StarlarkRunner {
	return &StarlarkRunner{}
}

// ModuleLoader can load starlark modules (like http)
type ModuleLoader func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// NewMockModuleLoader returns a ModuleLoader to load mock modules
func NewMockModuleLoader(proxy *proxyDetails) ModuleLoader {
	return func(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
		m := &MockHTTPModule{proxy: proxy}
		// TODO(dlong): Add other mock implementations, as needed
		return starlark.StringDict{
			"http": m.Struct(),
		}, nil
	}
}

// MockHTTPModule is a module for mocking out http functionality.
type MockHTTPModule struct {
	proxy *proxyDetails
}

// Struct returns a starlark struct with methods
func (m *MockHTTPModule) Struct() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlarkstruct.Default, m.StringDict())
}

// StringDict returns the module as a dictionary keyed by strings
func (m *MockHTTPModule) StringDict() starlark.StringDict {
	return starlark.StringDict{
		"get": starlark.NewBuiltin("get", m.get),
	}
}

// get performs a mock http request
func (m *MockHTTPModule) get(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if m.proxy != nil {
		// TODO: Check get's argument against m.proxy.URL
		result, err := starutil.Marshal(m.proxy.Response)
		if err != nil {
			return starlark.None, err
		}
		return result, nil
	}
	return starlark.None, fmt.Errorf("Cannot use http.get without WebProxy")
}

// Run runs the actual starlark code from a test case.
func (r *StarlarkRunner) Run(details *testDetails, sourceCode string) error {
	// Log information about the test before running it (debug level only).
	log.Debugf("==============================")
	log.Debugf("WebProxy: %p", details.WebProxy)
	log.Debugf("Setup:  %s", details.Setup)
	log.Debugf("Call:   %s", details.Call)
	log.Debugf("Actual: %s", details.Actual)
	log.Debugf("Expect: %s", details.Expect)
	log.Debugf("code: {%s}", sourceCode)
	log.Debugf("------------------------------")

	var err error
	thread := &starlark.Thread{
		Load: NewMockModuleLoader(details.WebProxy),
	}
	// Environment has `ds` and `ctx` predefined.
	environment := make(map[string]starlark.Value)
	ds := stards.Dataset{}
	ds.SetMutable(&dataset.Dataset{})
	ctx := context.NewContext(make(map[string]interface{}), make(map[string]interface{}))

	environment["ds"] = ds.Methods()
	environment["ctx"] = ctx.Struct()
	// Setup is either modifying a field of context (handled specially), or mutates an existing
	// variable (such as calling set_body on ds). It won't modify the top-level environment.
	if details.Setup != "" {
		log.Info("running Setup...")
		if strings.HasPrefix(details.Setup, "ctx.download = ") {
			// TODO: Actual implementation
			result := starlark.NewList([]starlark.Value{starlark.Value(starlark.String("test"))})
			ctx.SetResult("download", result)
		} else {
			_, err = starlark.ExecFile(thread, "", details.Setup, environment)
			if err != nil {
				return fmt.Errorf("during Setup: %s", err.Error())
			}
		}
	}

	environment["ds"] = ds.Methods()
	environment["ctx"] = ctx.Struct()
	// Run sourceCode
	log.Info("running code block...")
	environment, err = starlark.ExecFile(thread, "", sourceCode, environment)
	if err != nil {
		return fmt.Errorf("running code block: %s", err.Error())
	}

	environment["ds"] = ds.Methods()
	environment["ctx"] = ctx.Struct()
	// Call is the entry point to run in order to exercise the test case.
	// TODO(dlong): Validate that this is a single function
	log.Info("running Call...")
	environment, err = starlark.ExecFile(thread, "", "result = "+details.Call, environment)
	if err != nil {
		return fmt.Errorf("during Call: %s", err.Error())
	}

	// Assign special function results to ctx field
	if strings.HasPrefix(details.Call, "download") {
		ctx.SetResult("download", environment["result"])
	}

	environment["ds"] = ds.Methods()
	environment["ctx"] = ctx.Struct()
	// Actual accesses the results of the test case.
	// TODO(dlong): Validate that this is an expression (should not have side-effects)
	log.Info("running Actual...")
	environment, err = starlark.ExecFile(thread, "", "result = "+details.Actual, environment)
	if err != nil {
		return fmt.Errorf("during Actual: %s", err.Error())
	}

	// Parse the results from Actual into a native data structure.
	var actual interface{}
	resultStr := environment["result"].String()
	err = json.Unmarshal([]byte(resultStr), &actual)
	if err != nil {
		return fmt.Errorf("parsing \"%s\": %s", resultStr, err.Error())
	}

	actualString := fmt.Sprintf("%s", actual)
	expectString := fmt.Sprintf("%s", details.Expect)

	// Compare actual results against the expected results, fail if different.
	if !reflect.DeepEqual(actualString, expectString) {
		tmpl := `test case failure
  actual: %s
  expect: %s`
		return fmt.Errorf(tmpl, actualString, expectString)
	}

	log.Info("success!")
	return nil
}
