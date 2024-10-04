// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package backupccl

import (
	"context"
	"fmt"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/ccl/multiregionccl/multiregionccltestutils"
	_ "github.com/cockroachdb/cockroach/pkg/cloud/impl" // register cloud storage providers
	"github.com/cockroachdb/cockroach/pkg/jobs"
	"github.com/cockroachdb/cockroach/pkg/jobs/jobspb"
	"github.com/cockroachdb/cockroach/pkg/multitenant/tenantcapabilities"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql"
	_ "github.com/cockroachdb/cockroach/pkg/sql/importer"
	"github.com/cockroachdb/cockroach/pkg/testutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/jobutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/skip"
	"github.com/cockroachdb/cockroach/pkg/testutils/sqlutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/testcluster"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
)

func TestBackupSharedProcessTenantNodeDown(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	ctx := context.Background()

	skip.UnderRace(t, "multi-node, multi-tenant test too slow under race")
	params := base.TestClusterArgs{
		ServerArgs: base.TestServerArgs{
			DisableDefaultTestTenant: true,
		},
	}
	params.ServerArgs.Knobs.JobsTestingKnobs = jobs.NewTestingKnobsWithShortIntervals()
	tc, hostDB, _, cleanup := backupRestoreTestSetupWithParams(t, multiNode, 0, /* numAccounts */
		InitManualReplication, params)
	defer cleanup()

	hostDB.Exec(t, "ALTER TENANT ALL SET CLUSTER SETTING sql.split_at.allow_for_secondary_tenant.enabled=true")
	hostDB.Exec(t, "ALTER TENANT ALL SET CLUSTER SETTING sql.scatter.allow_for_secondary_tenant.enabled=true")
	hostDB.Exec(t, "ALTER TENANT ALL SET CLUSTER SETTING server.sqlliveness.ttl='2s'")
	hostDB.Exec(t, "ALTER TENANT ALL SET CLUSTER SETTING server.sqlliveness.heartbeat='250ms'")

	testTenantID := roachpb.MustMakeTenantID(11)
	_, tenantDB, err := tc.Server(0).StartSharedProcessTenant(ctx,
		base.TestSharedProcessTenantArgs{
			TenantID:   testTenantID,
			TenantName: "test",
			Knobs:      base.TestingKnobs{JobsTestingKnobs: jobs.NewTestingKnobsWithShortIntervals()},
		})
	require.NoError(t, err)

	hostDB.Exec(t, "ALTER TENANT test GRANT ALL CAPABILITIES")
	tc.WaitForTenantCapabilities(t, testTenantID, map[tenantcapabilities.ID]string{
		tenantcapabilities.CanUseNodelocalStorage: "true",
		tenantcapabilities.CanAdminSplit:          "true",
		tenantcapabilities.CanAdminScatter:        "true",
	})
	require.NoError(t, err)

	tenantSQL := sqlutils.MakeSQLRunner(tenantDB)
	tenantSQL.Exec(t, "CREATE TABLE foo AS SELECT generate_series(1, 4000)")
	tenantSQL.Exec(t, "ALTER TABLE foo SPLIT AT VALUES (500), (1000), (1500), (2000), (2500), (3000)")
	tenantSQL.Exec(t, "ALTER TABLE foo SCATTER")

	t.Log("waiting for SQL instances")
	waitStart := timeutil.Now()
	for i := 1; i < multiNode; i++ {
		sqlAddr := tc.Server(i).ServingSQLAddr()
		testutils.SucceedsSoon(t, func() error {
			t.Logf("waiting for server %d", i)
			db, err := serverutils.OpenDBConnE(sqlAddr, "cluster:test/defaultdb", false, tc.Stopper())
			if err != nil {
				return err
			}
			return db.Ping()
		})
	}
	t.Logf("all SQL instances (took %s)", timeutil.Since(waitStart))

	// Shut down a node.
	t.Log("shutting down server 2 (n3)")
	tc.StopServer(2)

	tenantRunner := sqlutils.MakeSQLRunner(tenantDB)
	var jobID jobspb.JobID
	tenantRunner.QueryRow(t, "BACKUP INTO 'nodelocal://1/worker-failure' WITH detached").Scan(&jobID)
	jobutils.WaitForJobToSucceed(t, tenantRunner, jobID)
}

func TestBackupTenantImportingTable(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	ctx := context.Background()
	tc := testcluster.StartTestCluster(t, 1,
		base.TestClusterArgs{
			ServerArgs: base.TestServerArgs{
				// Test is designed to run with explicit tenants. No need to
				// implicitly create a tenant.
				DisableDefaultTestTenant: true,
			},
		})
	defer tc.Stopper().Stop(ctx)
	sqlDB := sqlutils.MakeSQLRunner(tc.Conns[0])

	tSrv, tSQL := serverutils.StartTenant(t, tc.Server(0), base.TestTenantArgs{
		TenantID:     roachpb.MustMakeTenantID(10),
		TestingKnobs: base.TestingKnobs{JobsTestingKnobs: jobs.NewTestingKnobsWithShortIntervals()},
	})
	defer tSQL.Close()
	runner := sqlutils.MakeSQLRunner(tSQL)

	if _, err := tSQL.Exec("SET CLUSTER SETTING jobs.debug.pausepoints = 'import.after_ingest';"); err != nil {
		t.Fatal(err)
	}
	if _, err := tSQL.Exec("CREATE TABLE x (id INT PRIMARY KEY, n INT, s STRING)"); err != nil {
		t.Fatal(err)
	}
	if _, err := tSQL.Exec("INSERT INTO x VALUES (1000, 1, 'existing')"); err != nil {
		t.Fatal(err)
	}
	if _, err := tSQL.Exec("IMPORT INTO x CSV DATA ('workload:///csv/bank/bank?rows=100&version=1.0.0')"); !testutils.IsError(err, "pause") {
		t.Fatal(err)
	}
	var jobID jobspb.JobID
	err := tSQL.QueryRow(`SELECT job_id FROM [show jobs] WHERE job_type = 'IMPORT'`).Scan(&jobID)
	require.NoError(t, err)
	jobutils.WaitForJobToPause(t, runner, jobID)

	// tenant now has a fully ingested, paused import, so back them up.
	const dst = "userfile:///t"
	if _, err := sqlDB.DB.ExecContext(ctx, `BACKUP TENANT 10 TO $1`, dst); err != nil {
		t.Fatal(err)
	}
	// Destroy the tenant, then restore it.
	tSrv.Stopper().Stop(ctx)
	if _, err := sqlDB.DB.ExecContext(ctx, "DROP TENANT [10] IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.DB.ExecContext(ctx, "RESTORE TENANT 10 FROM $1", dst); err != nil {
		t.Fatal(err)
	}
	_, tSQL = serverutils.StartTenant(t, tc.Server(0), base.TestTenantArgs{
		TenantID:     roachpb.MustMakeTenantID(10),
		TestingKnobs: base.TestingKnobs{JobsTestingKnobs: jobs.NewTestingKnobsWithShortIntervals()},
	})
	defer tSQL.Close()

	if _, err := tSQL.Exec(`UPDATE system.jobs SET claim_session_id = NULL, claim_instance_id = NULL WHERE id = $1`, jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := tSQL.Exec(`DELETE FROM system.lease`); err != nil {
		t.Fatal(err)
	}
	testutils.SucceedsSoon(t, func() error {
		if _, err := tSQL.Exec(`CANCEL JOB $1`, jobID); err != nil {
			return err
		}

		var status string
		if err := tSQL.QueryRow(`SELECT status FROM [show jobs] WHERE job_id = $1`, jobID).Scan(&status); err != nil {
			return err
		}
		if status == string(jobs.StatusCanceled) {
			return nil
		}
		return errors.Newf("%s", status)
	})

	var rowCount int
	if err := tSQL.QueryRow(`SELECT count(*) FROM x`).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	require.Equal(t, 1, rowCount)
}

// TestTenantBackupMultiRegionDatabases ensures secondary tenants restoring
// MR databases respect the
// sql.multi_region.allow_abstractions_for_secondary_tenants.enabled cluster
// setting.
func TestTenantBackupMultiRegionDatabases(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	skip.UnderStressRace(t, "test is too heavy to run under stress")

	tc, db, cleanup := multiregionccltestutils.TestingCreateMultiRegionCluster(
		t, 3 /*numServers*/, base.TestingKnobs{},
	)
	defer cleanup()
	sqlDB := sqlutils.MakeSQLRunner(db)

	tenID := roachpb.MustMakeTenantID(10)
	_, tSQL := serverutils.StartTenant(t, tc.Server(0), base.TestTenantArgs{
		TenantID:     tenID,
		TestingKnobs: base.TestingKnobs{JobsTestingKnobs: jobs.NewTestingKnobsWithShortIntervals()},
	})
	defer tSQL.Close()
	tenSQLDB := sqlutils.MakeSQLRunner(tSQL)

	setAndWaitForTenantReadOnlyClusterSetting(
		t,
		sql.SecondaryTenantsMultiRegionAbstractionsEnabledSettingName,
		sqlDB,
		tenSQLDB,
		tenID,
		"true",
	)

	// Setup.
	const tenDst = "userfile:///ten_backup"
	const hostDst = "userfile:///host_backup"
	tenSQLDB.Exec(t, `CREATE DATABASE mrdb PRIMARY REGION "us-east1"`)
	tenSQLDB.Exec(t, fmt.Sprintf("BACKUP DATABASE mrdb INTO '%s'", tenDst))

	sqlDB.Exec(t, `CREATE DATABASE mrdb PRIMARY REGION "us-east1"`)
	sqlDB.Exec(t, fmt.Sprintf("BACKUP DATABASE mrdb INTO '%s'", hostDst))

	{
		// Flip the tenant-read only cluster setting; ensure database can be restored
		// on the system tenant but not on the secondary tenant.
		setAndWaitForTenantReadOnlyClusterSetting(
			t,
			sql.SecondaryTenantsMultiRegionAbstractionsEnabledSettingName,
			sqlDB,
			tenSQLDB,
			tenID,
			"false",
		)

		tenSQLDB.Exec(t, "DROP DATABASE mrdb CASCADE")
		tenSQLDB.ExpectErr(
			t,
			"setting .* disallows secondary tenant to restore a multi-region database",
			fmt.Sprintf("RESTORE DATABASE mrdb FROM LATEST IN '%s'", tenDst),
		)

		// The system tenant should remain unaffected.
		sqlDB.Exec(t, "DROP DATABASE mrdb CASCADE")
		sqlDB.Exec(t, fmt.Sprintf("RESTORE DATABASE mrdb FROM LATEST IN '%s'", hostDst))
	}

	{
		// Flip the tenant-read only cluster setting back to true and ensure the
		// restore succeeds.
		setAndWaitForTenantReadOnlyClusterSetting(
			t,
			sql.SecondaryTenantsMultiRegionAbstractionsEnabledSettingName,
			sqlDB,
			tenSQLDB,
			tenID,
			"true",
		)

		tenSQLDB.Exec(t, fmt.Sprintf("RESTORE DATABASE mrdb FROM LATEST IN '%s'", tenDst))
	}

	{
		// Ensure tenant's restoring non multi-region databases are unaffected
		// by this setting. We set sql.defaults.primary_region for good measure.
		tenSQLDB.Exec(
			t,
			fmt.Sprintf(
				"SET CLUSTER SETTING %s = 'us-east1'", sql.DefaultPrimaryRegionClusterSettingName,
			),
		)
		setAndWaitForTenantReadOnlyClusterSetting(
			t,
			sql.SecondaryTenantsMultiRegionAbstractionsEnabledSettingName,
			sqlDB,
			tenSQLDB,
			tenID,
			"false",
		)

		tenSQLDB.Exec(t, "CREATE DATABASE nonMrDB")
		tenSQLDB.Exec(t, fmt.Sprintf("BACKUP DATABASE nonMrDB INTO '%s'", tenDst))

		tenSQLDB.Exec(t, "DROP DATABASE nonMrDB CASCADE")
		tenSQLDB.Exec(t, fmt.Sprintf("RESTORE DATABASE nonMrDB FROM LATEST IN '%s'", tenDst))
	}
}
