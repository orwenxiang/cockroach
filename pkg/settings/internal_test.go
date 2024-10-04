// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

var b = RegisterBoolSetting(SystemOnly, "b", "desc", true)

func TesIgnoreDefaults(t *testing.T) {
	ctx := context.Background()
	sv := &Values{}
	sv.Init(ctx, TestOpaque)

	ignoreAllUpdates = true
	defer func() { ignoreAllUpdates = false }()
	u := NewUpdater(sv)
	require.NoError(t, u.Set(ctx, b.Key(), EncodedValue{Value: EncodeBool(false), Type: "b"}))
	require.Equal(t, true, b.Get(sv))

	ignoreAllUpdates = false
	u = NewUpdater(sv)
	require.NoError(t, u.Set(ctx, b.Key(), EncodedValue{Value: EncodeBool(false), Type: "b"}))
	require.Equal(t, false, b.Get(sv))
}
