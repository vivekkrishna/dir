// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"time"
)

type Annotation struct {
	ID        uint   `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	Key       string `gorm:"not null"`
	Value     string `gorm:"not null"`
}

// convertAnnotations transforms a map of annotation key-value pairs to Database Annotation structs.
func convertAnnotations(annotations map[string]string, recordCID string) []Annotation {
	result := make([]Annotation, 0, len(annotations))
	for key, value := range annotations {
		result = append(result, Annotation{
			RecordCID: recordCID,
			Key:       key,
			Value:     value,
		})
	}

	return result
}
