package constants

import "testing"

func TestMeasurementStateMachine(t *testing.T) {
	if !CanTransitionMeasurement(MeasurementCaptured, MeasurementValidated) {
		t.Fatal("captured -> validated must be legal")
	}
	if CanTransitionMeasurement(MeasurementCaptured, MeasurementReady) {
		t.Fatal("captured -> ready must be rejected")
	}
	if !CanTransitionMeasurement(MeasurementReady, MeasurementSuperseded) {
		t.Fatal("ready -> superseded must be legal")
	}
}

func TestAttributionStateMachine(t *testing.T) {
	if !CanTransitionAttribution(AttributionCompleted, AttributionReviewed) {
		t.Fatal("completed -> reviewed must be legal")
	}
	if CanTransitionAttribution(AttributionCompleted, AttributionConfirmed) {
		t.Fatal("completed -> confirmed must require independent review")
	}
	if CanTransitionAttribution(AttributionConfirmed, AttributionVoided) {
		t.Fatal("confirmed result must remain immutable")
	}
}

func TestProfileStateMachine(t *testing.T) {
	if !CanTransitionProfile(ProfileDraft, ProfileActive) {
		t.Fatal("draft -> active must be legal")
	}
	if !CanTransitionProfile(ProfileActive, ProfileRetired) {
		t.Fatal("active -> retired must be legal")
	}
	if !CanTransitionProfile(ProfileRetired, ProfileActive) {
		t.Fatal("retired -> active must allow switching an old version back")
	}
	if CanTransitionProfile(ProfileDraft, ProfileRetired) {
		t.Fatal("draft -> retired must be rejected")
	}
}
