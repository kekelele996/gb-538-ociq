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
