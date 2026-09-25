package constants

type AttributionState string

const (
	AttributionQueued      AttributionState = "queued"
	AttributionCalculating AttributionState = "calculating"
	AttributionCompleted   AttributionState = "completed"
	AttributionFailed      AttributionState = "failed"
	AttributionReviewed    AttributionState = "reviewed"
	AttributionConfirmed   AttributionState = "confirmed"
	AttributionVoided      AttributionState = "voided"
)

var AttributionTransitions = map[AttributionState]map[AttributionState]bool{
	AttributionQueued:      {AttributionCalculating: true},
	AttributionCalculating: {AttributionCompleted: true, AttributionFailed: true},
	AttributionCompleted:   {AttributionReviewed: true, AttributionVoided: true},
	AttributionReviewed:    {AttributionConfirmed: true, AttributionVoided: true},
}

func CanTransitionAttribution(from, to AttributionState) bool {
	return AttributionTransitions[from][to]
}
