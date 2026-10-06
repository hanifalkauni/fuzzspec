package reporter

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/oracle"
)

type JUnitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// ExportJUnit writes JUnit XML test results to disk.
func ExportJUnit(filePath string, assertions []oracle.AssertionResult, totalDuration time.Duration) error {
	var failures int
	var testCases []JUnitTestCase

	for _, a := range assertions {
		durationSec := fmt.Sprintf("%.4f", a.Execution.Duration.Seconds())
		className := strings.TrimPrefix(a.Vector.Path, "/")
		if className == "" {
			className = "root"
		}
		caseName := fmt.Sprintf("%s %s - %s", a.Vector.Method, a.Vector.Path, a.Vector.Scenario)

		tc := JUnitTestCase{
			Name:      caseName,
			ClassName: className,
			Time:      durationSec,
		}

		if a.Verdict == oracle.VerdictFail {
			failures++
			findingMsg := "Assertion failed"
			if len(a.Findings) > 0 {
				findingMsg = a.Findings[0].Message
			}
			tc.Failure = &JUnitFailure{
				Message: findingMsg,
				Type:    "HTTP500CrashOrContractDrift",
				Content: fmt.Sprintf("Target: %s\nStatus: %d %s\nScenario: %s\nReproducer: %s\nResponse: %s",
					a.Execution.RequestURL,
					a.Execution.StatusCode,
					a.Execution.StatusText,
					a.Vector.Scenario,
					a.Execution.CurlCommand,
					a.Execution.ResponseBody,
				),
			}
		}

		testCases = append(testCases, tc)
	}

	totalSec := fmt.Sprintf("%.4f", totalDuration.Seconds())
	suites := JUnitTestSuites{
		Name:     "FuzzSpec Test Suite",
		Tests:    len(assertions),
		Failures: failures,
		Errors:   0,
		Time:     totalSec,
		Suites: []JUnitTestSuite{
			{
				Name:      "Spec-to-Contract Fuzzing",
				Tests:     len(assertions),
				Failures:  failures,
				Errors:    0,
				Time:      totalSec,
				TestCases: testCases,
			},
		},
	}

	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	fullXML := []byte(xml.Header + string(data))
	sanitized := []byte(SanitizeText(string(fullXML)))

	return os.WriteFile(filePath, sanitized, 0644)
}
