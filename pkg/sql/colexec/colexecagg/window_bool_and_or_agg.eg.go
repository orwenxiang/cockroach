// Code generated by execgen; DO NOT EDIT.
// Copyright 2020 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package colexecagg

import (
	"unsafe"

	"github.com/cockroachdb/cockroach/pkg/col/coldata"
	"github.com/cockroachdb/cockroach/pkg/sql/colexecerror"
	"github.com/cockroachdb/cockroach/pkg/sql/colmem"
	"github.com/cockroachdb/errors"
)

// Remove unused warning.
var _ = colexecerror.InternalError

func newBoolAndWindowAggAlloc(
	allocator *colmem.Allocator, allocSize int64,
) aggregateFuncAlloc {
	return &boolAndWindowAggAlloc{aggAllocBase: aggAllocBase{
		allocator: allocator,
		allocSize: allocSize,
	}}
}

type boolAndWindowAgg struct {
	unorderedAggregateFuncBase
	curAgg bool
	// foundNonNullForCurrentGroup tracks if we have seen any non-null values
	// for the group that is currently being aggregated.
	foundNonNullForCurrentGroup bool
}

var _ AggregateFunc = &boolAndWindowAgg{}

func (a *boolAndWindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	var oldCurAggSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Bool(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				a.curAgg = a.curAgg && col[i]
				a.foundNonNullForCurrentGroup = true
			}

		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				a.curAgg = a.curAgg && col[i]
				a.foundNonNullForCurrentGroup = true
			}

		}
	}
	var newCurAggSize uintptr
	if newCurAggSize != oldCurAggSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurAggSize - oldCurAggSize))
	}
}

func (a *boolAndWindowAgg) Flush(outputIdx int) {
	col := a.vec.Bool()
	if !a.foundNonNullForCurrentGroup {
		a.nulls.SetNull(outputIdx)
	} else {
		col[outputIdx] = a.curAgg
	}
}

func (a *boolAndWindowAgg) Reset() {
	a.curAgg = true
	a.foundNonNullForCurrentGroup = false
}

type boolAndWindowAggAlloc struct {
	aggAllocBase
	aggFuncs []boolAndWindowAgg
}

var _ aggregateFuncAlloc = &boolAndWindowAggAlloc{}

const sizeOfBoolAndWindowAgg = int64(unsafe.Sizeof(boolAndWindowAgg{}))
const boolAndWindowAggSliceOverhead = int64(unsafe.Sizeof([]boolAndWindowAgg{}))

func (a *boolAndWindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(boolAndWindowAggSliceOverhead + sizeOfBoolAndWindowAgg*a.allocSize)
		a.aggFuncs = make([]boolAndWindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	f.Reset()
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go). This allows bool_and and bool_or operators to be
// used when the window frame only grows. For the case when the window frame can
// shrink, the default quadratic-scaling implementation is necessary.
func (*boolAndWindowAgg) Remove(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int,
) {
	colexecerror.InternalError(
		errors.AssertionFailedf("Remove called on boolAndWindowAgg"),
	)
}

func newBoolOrWindowAggAlloc(
	allocator *colmem.Allocator, allocSize int64,
) aggregateFuncAlloc {
	return &boolOrWindowAggAlloc{aggAllocBase: aggAllocBase{
		allocator: allocator,
		allocSize: allocSize,
	}}
}

type boolOrWindowAgg struct {
	unorderedAggregateFuncBase
	curAgg bool
	// foundNonNullForCurrentGroup tracks if we have seen any non-null values
	// for the group that is currently being aggregated.
	foundNonNullForCurrentGroup bool
}

var _ AggregateFunc = &boolOrWindowAgg{}

func (a *boolOrWindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	var oldCurAggSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Bool(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				a.curAgg = a.curAgg || col[i]
				a.foundNonNullForCurrentGroup = true
			}

		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				a.curAgg = a.curAgg || col[i]
				a.foundNonNullForCurrentGroup = true
			}

		}
	}
	var newCurAggSize uintptr
	if newCurAggSize != oldCurAggSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurAggSize - oldCurAggSize))
	}
}

func (a *boolOrWindowAgg) Flush(outputIdx int) {
	col := a.vec.Bool()
	if !a.foundNonNullForCurrentGroup {
		a.nulls.SetNull(outputIdx)
	} else {
		col[outputIdx] = a.curAgg
	}
}

func (a *boolOrWindowAgg) Reset() {
	a.curAgg = false
	a.foundNonNullForCurrentGroup = false
}

type boolOrWindowAggAlloc struct {
	aggAllocBase
	aggFuncs []boolOrWindowAgg
}

var _ aggregateFuncAlloc = &boolOrWindowAggAlloc{}

const sizeOfBoolOrWindowAgg = int64(unsafe.Sizeof(boolOrWindowAgg{}))
const boolOrWindowAggSliceOverhead = int64(unsafe.Sizeof([]boolOrWindowAgg{}))

func (a *boolOrWindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(boolOrWindowAggSliceOverhead + sizeOfBoolOrWindowAgg*a.allocSize)
		a.aggFuncs = make([]boolOrWindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	f.Reset()
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go). This allows bool_and and bool_or operators to be
// used when the window frame only grows. For the case when the window frame can
// shrink, the default quadratic-scaling implementation is necessary.
func (*boolOrWindowAgg) Remove(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int,
) {
	colexecerror.InternalError(
		errors.AssertionFailedf("Remove called on boolOrWindowAgg"),
	)
}
