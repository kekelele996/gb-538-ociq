package constants

type MeasurementQuality string

const (
	QualityValid        MeasurementQuality = "valid"
	QualityContaminated MeasurementQuality = "contaminated"
	QualityClipped      MeasurementQuality = "clipped"
	QualityMissing      MeasurementQuality = "missing"
)

var MeasurementQualities = []MeasurementQuality{
	QualityValid,
	QualityContaminated,
	QualityClipped,
	QualityMissing,
}

func IsMeasurementQuality(value string) bool {
	for _, candidate := range MeasurementQualities {
		if string(candidate) == value {
			return true
		}
	}
	return false
}
