// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/testutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/datapathutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/skip"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/datadriven"
)

// TestTenantZip tests the operation of zip for a tenant server.
func TestTenantZip(t *testing.T) {
	defer leaktest.AfterTest(t)()
	skip.UnderRace(t, "test too slow under race")

	tenants := []struct {
		testName      string
		addTenantArgs func(params TestCLIParams) TestCLIParams
	}{
		{
			testName: "testzip tenant separate process",
			addTenantArgs: func(params TestCLIParams) TestCLIParams {
				tenantDir, tenantDirCleanupFn := testutils.TempDir(t)
				defer tenantDirCleanupFn()
				params.TenantArgs = &base.TestTenantArgs{
					TenantID:             serverutils.TestTenantID(),
					HeapProfileDirName:   tenantDir,
					GoroutineDumpDirName: tenantDir,
				}
				return params
			},
		},
		{
			testName: "testzip shared process tenant",
			addTenantArgs: func(params TestCLIParams) TestCLIParams {
				params.SharedProcessTenantArgs = &base.TestSharedProcessTenantArgs{
					TenantName: "test-tenant",
					TenantID:   serverutils.TestTenantID(),
				}
				params.UseSystemTenant = true
				return params
			},
		},
	}

	for _, tenant := range tenants {
		t.Run(tenant.testName, func(t *testing.T) {
			hostDir, hostDirCleanupFn := testutils.TempDir(t)
			defer hostDirCleanupFn()
			c := NewCLITest(tenant.addTenantArgs(TestCLIParams{
				StoreSpecs: []base.StoreSpec{{
					Path: hostDir,
				}},
				// TODO(abarganier): Switch to secure mode once underlying infra has been
				// updated to support it. See: https://github.com/cockroachdb/cockroach/issues/77173
				Insecure: true,
			}))
			defer c.Cleanup()

			out, err := c.RunWithCapture("debug zip --concurrency=1 --cpu-profile-duration=1s " + os.DevNull)
			if err != nil {
				t.Fatal(err)
			}

			// Strip any non-deterministic messages.
			out = eraseNonDeterministicZipOutput(out)

			// We use datadriven simply to read the golden output file; we don't actually
			// run any commands. Using datadriven allows TESTFLAGS=-rewrite.
			datadriven.RunTest(t,
				datapathutils.TestDataPath(t, "zip", strings.ReplaceAll(tenant.testName, " ", "_")),
				func(t *testing.T, td *datadriven.TestData) string {
					return out
				},
			)
		})
	}
}
