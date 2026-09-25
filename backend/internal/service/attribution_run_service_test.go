package service

import (
	"testing"
	"time"

	"industrial-noise-source-attribution/backend/internal/model"
)

func TestAttributionResponseKeepsEqualIDArrays(t *testing.T) {
	run := model.AttributionRun{
		ID: 1, RunCode: "AR-TEST", MeasurementIDsJSON: "[1,2,3]", SourceProfileIDsJSON: "[1,2,3]",
		AlgorithmVersion: "test", InputHash: "hash", InputSnapshotJSON: "{}", NormalizedBandsJSON: "[]",
		ContributionsJSON: "[]", EvidenceJSON: "{}", AttributionState: "completed", StartedAt: time.Now(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	response, err := attributionResponse(run)
	if err != nil {
		t.Fatalf("attributionResponse returned error: %v", err)
	}
	if len(response.MeasurementIDs) != 3 || len(response.SourceProfileIDs) != 3 {
		t.Fatalf("equal JSON arrays overwrote each other: measurements=%v sources=%v", response.MeasurementIDs, response.SourceProfileIDs)
	}
}
