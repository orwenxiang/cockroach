// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

import * as protos from "@cockroachlabs/crdb-protobuf-client";
import { AggregateStatistics } from "src/statementsTable";
import { longToInt } from "./fixLong";

type Statement =
  protos.cockroach.server.serverpb.StatementsResponse.ICollectedStatementStatistics;
type statementType = AggregateStatistics | Statement;
type statementsType = Array<statementType>;

/**
 * Function to calculate total workload of statements
 * Currently is recalculating every time is called, if that becomes an issue
 * on the future, consider use of cache and memoize the function
 * @param statements array of statements (AggregateStatistics or Statement)
 * @returns the total workload of all statements
 */
export function calculateTotalWorkload(statements: statementsType): number {
  return statements.reduce((totalWorkload: number, stmt: statementType) => {
    return (totalWorkload +=
      longToInt(stmt.stats.count) * stmt.stats.service_lat.mean);
  }, 0);
}
