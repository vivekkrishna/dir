// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryToFilters_AnnotationKeyValue(t *testing.T) {
	queries := []*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "team:platform"},
	}

	options, err := QueryToFilters(queries)
	require.NoError(t, err)
	assert.Len(t, options, 2) // key + value
}

func TestQueryToFilters_AnnotationKeyOnly(t *testing.T) {
	queries := []*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "owner"},
	}

	options, err := QueryToFilters(queries)
	require.NoError(t, err)
	assert.Len(t, options, 1) // key only
}

func TestQueryToFilters_AnnotationEmptyKey(t *testing.T) {
	queries := []*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: ":somevalue"},
	}

	options, err := QueryToFilters(queries)
	require.NoError(t, err)
	assert.Len(t, options, 0) // skipped due to empty key
}

func TestQueryToFilters_AnnotationKeyWithEmptyValue(t *testing.T) {
	queries := []*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "team:"},
	}

	options, err := QueryToFilters(queries)
	require.NoError(t, err)
	assert.Len(t, options, 1) // key only, empty value skipped
}

func TestQueryToFilters_AnnotationWildcard(t *testing.T) {
	queries := []*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "env:prod*"},
	}

	options, err := QueryToFilters(queries)
	require.NoError(t, err)
	assert.Len(t, options, 2) // key + wildcard value
}
