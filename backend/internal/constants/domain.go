package constants

type MeasurementState string

const (
	MeasurementCaptured   MeasurementState = "captured"
	MeasurementValidated  MeasurementState = "validated"
	MeasurementNormalized MeasurementState = "normalized"
	MeasurementReady      MeasurementState = "ready"
	MeasurementRejected   MeasurementState = "rejected"
	MeasurementSuperseded MeasurementState = "superseded"
)

var MeasurementTransitions = map[MeasurementState]map[MeasurementState]bool{
	MeasurementCaptured:   {MeasurementValidated: true, MeasurementRejected: true},
	MeasurementValidated:  {MeasurementNormalized: true, MeasurementRejected: true},
	MeasurementNormalized: {MeasurementReady: true, MeasurementRejected: true},
	MeasurementReady:      {MeasurementSuperseded: true},
}

func CanTransitionMeasurement(from, to MeasurementState) bool {
	return MeasurementTransitions[from][to]
}

type PointState string

const (
	PointActive   PointState = "active"
	PointInactive PointState = "inactive"
)

type ProfileState string

const (
	ProfileDraft   ProfileState = "draft"
	ProfileActive  ProfileState = "active"
	ProfileRetired ProfileState = "retired"
)

var ProfileTransitions = map[ProfileState]map[ProfileState]bool{
	ProfileDraft:  {ProfileActive: true},
	ProfileActive: {ProfileRetired: true},
}

func CanTransitionProfile(from, to ProfileState) bool {
	return ProfileTransitions[from][to]
}

const (
	RoleAdmin            = "admin"
	RoleAcousticEngineer = "acoustic_engineer"
	RoleDataAnalyst      = "data_analyst"
	RoleReviewer         = "reviewer"
	RoleAuditor          = "auditor"
)

var Roles = []string{RoleAdmin, RoleAcousticEngineer, RoleDataAnalyst, RoleReviewer, RoleAuditor}

const AlgorithmVersion = "octave-nnls-v1.0.0"

var OctaveBands = []int{63, 125, 250, 500, 1000, 2000, 4000, 8000}
