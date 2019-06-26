package framework

// DocrunFixture is a wrapper that doesn't do much aside from cause metadata blocks to need to
// begin with the text "docrun:" so that it's obvious that they're using docrun.
type DocrunFixture struct {
	Docrun docrunDetails
}

// docrunDetails holds all the metadata about the source code that follows it.
type docrunDetails struct {
	// Only one of the following three fields should be specified
	Pass    bool
	Test    *testDetails
	Command *commandDetails
	// Type of data structure to fill
	Filltype string
	// These two fields are entirely optional
	Lang string
	Save *saveDetails
}

// testDetails holds metadata about a test case. Expected results are recorded in this structure.
type testDetails struct {
	WebProxy *proxyDetails
	Setup    string
	Call     string
	Actual   string
	Expect   interface{}
}

// proxyDetails is used for tests that need a mock http response
type proxyDetails struct {
	URL      string
	Response interface{}
}

// saveDetails is used to save source code to a file for future commands
type saveDetails struct {
	Filename string
	Append   bool
}

// DocrunSource is parsed source code, which may have a specified language
type DocrunSource struct {
	Code string
	Lang string
}

// commandDetails holds information about commands to run
type commandDetails struct {
	SnapshotID string `json:"snapshotid"`
}
