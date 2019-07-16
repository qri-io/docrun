package framework

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/startf/context"
	stards "github.com/qri-io/qri/startf/ds"
	starhtml "github.com/qri-io/starlib/html"
	starhttp "github.com/qri-io/starlib/http"
	startime "github.com/qri-io/starlib/time"
	starutil "github.com/qri-io/starlib/util"
	starxlsx "github.com/qri-io/starlib/xlsx"
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
		if module == "http.star" {
			m := &MockHTTPModule{proxy: proxy}
			return starlark.StringDict{
				"http": m.Struct(),
			}, nil
		} else if module == "qri.star" {
			m := &MockQriModule{}
			return starlark.StringDict{
				"qri": m.Struct(),
			}, nil
		} else if module == "html.star" {
			return starhtml.LoadModule()
		} else if module == "time.star" {
			return startime.LoadModule()
		} else if module == "xlsx.star" {
			return starxlsx.LoadModule()
		} else {
			return nil, fmt.Errorf("module not defined: \"%s\"", module)
		}
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
		// Construct a fake http response.
		rec := httptest.NewRecorder()
		rec.WriteString(result.String())
		res := rec.Result()
		r := &starhttp.Response{*res}
		// Attack the request, convert to starlark struct type.
		r.Request = httptest.NewRequest("GET", m.proxy.URL, nil)
		return r.Struct(), nil
	}
	return starlark.None, fmt.Errorf("Cannot use http.get without WebProxy")
}

// MockQriModule is a module for mocking out qri functionality.
type MockQriModule struct {
}

// Struct returns a starlark struct with methods
func (m *MockQriModule) Struct() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlarkstruct.Default, m.StringDict())
}

// StringDict returns the module as a dictionary keyed by strings
func (m *MockQriModule) StringDict() starlark.StringDict {
	return starlark.StringDict{
		"list_datasets": starlark.NewBuiltin("list_datasets", m.listDatasets),
	}
}

// ListDatasets creates a list of mock datasetrefs
func (m *MockQriModule) listDatasets(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ls := &starlark.List{}
	ls.Append(starlark.String("test/ds_1@QmExample/ipfs/QmExample"))
	ls.Append(starlark.String("test/ds_2@QmSample/ipfs/QmSample"))
	return ls, nil
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

	// Preserve stdout so it can be captured
	stdoutTempFile := filepath.Join(os.TempDir(), "stdout")
	captureWrite, _ := os.Create(stdoutTempFile)
	preserveOut := os.Stdout

	environment["ds"] = ds.Methods()
	environment["ctx"] = ctx.Struct()
	// Call is the entry point to run in order to exercise the test case.
	// TODO(dlong): Validate that this is a single function
	log.Info("running Call...")
	// Capture stdout when the main part is executed
	// TODO: `print` statement in starlark is writing to stderr, not stdout. Fix this, please.
	os.Stderr = captureWrite
	environment, err = starlark.ExecFile(thread, "", "result = "+details.Call, environment)
	captureWrite.Close()
	os.Stderr = preserveOut
	if err != nil {
		return fmt.Errorf("during Call: %s", err.Error())
	}

	// Assign special function results to ctx field
	if strings.HasPrefix(details.Call, "download") {
		ctx.SetResult("download", environment["result"])
	}

	// Actual accesses the results of the test case.
	log.Info("running Actual...")
	var actual interface{}

	if details.Actual == "" {
		log.Info("blank Actual, nothing to do")
		log.Info("success!")
		return nil
	} else if details.Actual == "stdout.get()" {
		// Get what was written to stdout.
		stdoutText, _ := ioutil.ReadFile(stdoutTempFile)
		actual = strings.TrimSpace(string(stdoutText))
	} else {
		environment["ds"] = ds.Methods()
		environment["ctx"] = ctx.Struct()
		// TODO(dlong): Validate that this is an expression (should not have side-effects)
		environment, err = starlark.ExecFile(thread, "", "result = "+details.Actual, environment)
		if err != nil {
			return fmt.Errorf("during Actual: %s", err.Error())
		}
		// Parse the results from Actual into a native data structure.
		resultStr := environment["result"].String()
		err = json.Unmarshal([]byte(resultStr), &actual)
		if err != nil {
			return fmt.Errorf("parsing \"%s\": %s", resultStr, err.Error())
		}
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
