// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

// Code generated by generate-logictest, DO NOT EDIT.

package testfakedist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/build/bazel"
	"github.com/cockroachdb/cockroach/pkg/security/securityassets"
	"github.com/cockroachdb/cockroach/pkg/security/securitytest"
	"github.com/cockroachdb/cockroach/pkg/server"
	"github.com/cockroachdb/cockroach/pkg/sql"
	"github.com/cockroachdb/cockroach/pkg/sql/logictest"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/skip"
	"github.com/cockroachdb/cockroach/pkg/testutils/testcluster"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/randutil"
)

const configIdx = 4

var execBuildLogicTestDir string

func init() {
	if bazel.BuiltWithBazel() {
		var err error
		execBuildLogicTestDir, err = bazel.Runfile("pkg/sql/opt/exec/execbuilder/testdata")
		if err != nil {
			panic(err)
		}
	} else {
		execBuildLogicTestDir = "../../../../../../sql/opt/exec/execbuilder/testdata"
	}
}

func TestMain(m *testing.M) {
	securityassets.SetLoader(securitytest.EmbeddedAssets)
	randutil.SeedForTests()
	serverutils.InitTestServerFactory(server.TestServerFactory)
	serverutils.InitTestClusterFactory(testcluster.TestClusterFactory)

	defer serverutils.TestingSetDefaultTenantSelectionOverride(
		base.TestIsForStuffThatShouldWorkWithSecondaryTenantsButDoesntYet(76378),
	)()

	os.Exit(m.Run())
}

func runExecBuildLogicTest(t *testing.T, file string) {
	defer sql.TestingOverrideExplainEnvVersion("CockroachDB execbuilder test version")()
	skip.UnderDeadlock(t, "times out and/or hangs")
	serverArgs := logictest.TestServerArgs{
		DisableWorkmemRandomization: true,
		ForceProductionValues:       true,
		// Disable the direct scans in order to keep the output of EXPLAIN (VEC)
		// deterministic.
		DisableDirectColumnarScans: true,
	}
	logictest.RunLogicTest(t, serverArgs, configIdx, filepath.Join(execBuildLogicTestDir, file))
}

// TestLogic_tmp runs any tests that are prefixed with "_", in which a dedicated
// test is not generated for. This allows developers to create and run temporary
// test files that are not checked into the repository, without repeatedly
// regenerating and reverting changes to this file, generated_test.go.
//
// TODO(mgartner): Add file filtering so that individual files can be run,
// instead of all files with the "_" prefix.
func TestLogic_tmp(t *testing.T) {
	defer leaktest.AfterTest(t)()
	var glob string
	glob = filepath.Join(execBuildLogicTestDir, "_*")
	serverArgs := logictest.TestServerArgs{
		DisableWorkmemRandomization: true,
	}
	logictest.RunLogicTests(t, serverArgs, configIdx, glob)
}

func TestExecBuild_geospatial(
	t *testing.T,
) {
	defer leaktest.AfterTest(t)()
	runExecBuildLogicTest(t, "geospatial")
}
