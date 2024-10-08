// Copyright 2020 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package log

import (
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log/logconfig"
	"github.com/cockroachdb/datadriven"
	"github.com/pmezard/go-difflib/difflib"
)

func TestAppliedStandaloneConfig(t *testing.T) {
	defer leaktest.AfterTest(t)()

	const expected = `sinks:
  stderr:
    channels: {INFO: all}
    format: crdb-v2-tty
    redact: false
    redactable: false
    exit-on-error: true
`
	actual := DescribeAppliedConfig()
	if expected != actual {
		t.Errorf("expected:\n%s\ngot:\n%s\ndiff:\n%s",
			expected, actual, getDiff(expected, actual))
	}
}

func getDiff(expected, actual string) string {
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})
	return diff
}

func TestAppliedConfig(t *testing.T) {
	defer leaktest.AfterTest(t)()
	sc := ScopeWithoutShowLogs(t)
	defer sc.Close(t)

	datadriven.RunTest(t, "testdata/config",
		func(t *testing.T, d *datadriven.TestData) string {
			// Load the default config and apply the test's input.
			h := logconfig.Holder{Config: logconfig.DefaultConfig()}
			if err := h.Set(d.Input); err != nil {
				t.Fatal(err)
			}
			if err := h.Config.Validate(&sc.logDir); err != nil {
				t.Fatal(err)
			}

			TestingResetActive()
			cleanup, err := ApplyConfig(h.Config, nil /* fileSinkMetricsForDir */, nil /* fatalOnLogStall */)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()

			actual := DescribeAppliedConfig()
			// Make the test output deterministic.
			actual = strings.ReplaceAll(actual, sc.logDir, "TMPDIR")
			return actual
		})
}
