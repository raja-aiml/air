package tests

import "fmt"

// TestingT is an interface that abstracts testing.T for use in both
// automated tests and manual command-line verification
type TestingT interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Failed() bool
}

// ManualTester implements TestingT for command-line output
type ManualTester struct {
	Verbose bool
	failed  bool
}

func NewManualTester(verbose bool) *ManualTester {
	return &ManualTester{Verbose: verbose}
}

func (m *ManualTester) Logf(format string, args ...interface{}) {
	if m.Verbose {
		fmt.Printf("  ℹ️  "+format+"\n", args...)
	}
}

func (m *ManualTester) Errorf(format string, args ...interface{}) {
	fmt.Printf("  ❌ "+format+"\n", args...)
	m.failed = true
}

func (m *ManualTester) Failed() bool {
	return m.failed
}
