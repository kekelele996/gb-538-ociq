package algorithm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"industrial-noise-source-attribution/backend/internal/constants"
)

type Spectrum map[string]float64

type PointInput struct {
	ID         uint     `json:"id"`
	PointCode  string   `json:"point_code"`
	XM         float64  `json:"x_m"`
	YM         float64  `json:"y_m"`
	HeightM    float64  `json:"height_m"`
	Background Spectrum `json:"background"`
}

type MeasurementInput struct {
	ID       uint       `json:"id"`
	Checksum string     `json:"checksum"`
	Point    PointInput `json:"point"`
	Bands    Spectrum   `json:"bands"`
}

type SourceInput struct {
	ID                 uint     `json:"id"`
	SourceCode         string   `json:"source_code"`
	Name               string   `json:"name"`
	XM                 float64  `json:"x_m"`
	YM                 float64  `json:"y_m"`
	HeightM            float64  `json:"height_m"`
	ReferenceDistanceM float64  `json:"reference_distance_m"`
	Power              Spectrum `json:"power"`
	Directivity        Spectrum `json:"directivity"`
	OperatingFactor    float64  `json:"operating_factor"`
	Version            uint     `json:"version"`
}

type BandResult struct {
	BandHz      int     `json:"band_hz"`
	PredictedDB float64 `json:"predicted_db"`
	EnergyShare float64 `json:"energy_share"`
}

type ContributionResult struct {
	SourceProfileID uint         `json:"source_profile_id"`
	SourceCode      string       `json:"source_code"`
	SourceName      string       `json:"source_name"`
	Coefficient     float64      `json:"coefficient"`
	ContributionPct float64      `json:"contribution_pct"`
	OverallDB       float64      `json:"overall_db"`
	Bands           []BandResult `json:"bands"`
}

type Evidence struct {
	MatrixRows      int      `json:"matrix_rows"`
	MatrixColumns   int      `json:"matrix_columns"`
	Iterations      int      `json:"iterations"`
	Converged       bool     `json:"converged"`
	ConditionHint   float64  `json:"condition_hint"`
	UnreliableBands []string `json:"unreliable_bands"`
	Warnings        []string `json:"warnings"`
	Objective       float64  `json:"objective"`
	ElapsedMillis   int64    `json:"elapsed_millis"`
}

type Result struct {
	NormalizedBands []Spectrum           `json:"normalized_bands"`
	Contributions   []ContributionResult `json:"contributions"`
	ResidualError   float64              `json:"residual_error"`
	Evidence        Evidence             `json:"evidence"`
	Explanation     string               `json:"explanation"`
	PredictedRows   []float64            `json:"predicted_rows"`
}

func BandKey(band int) string { return strconv.Itoa(band) }

func ValidateSpectrum(s Spectrum, label string) error {
	if len(s) != len(constants.OctaveBands) {
		return fmt.Errorf("%s must contain exactly %d octave bands", label, len(constants.OctaveBands))
	}
	for _, band := range constants.OctaveBands {
		value, ok := s[BandKey(band)]
		if !ok {
			return fmt.Errorf("%s is missing %d Hz", label, band)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 180 {
			return fmt.Errorf("%s %d Hz must be finite and between 0 and 180 dB", label, band)
		}
	}
	return nil
}

func ValidateDirectivity(s Spectrum) error {
	if len(s) != len(constants.OctaveBands) {
		return fmt.Errorf("directivity must contain exactly %d octave bands", len(constants.OctaveBands))
	}
	for _, band := range constants.OctaveBands {
		value, ok := s[BandKey(band)]
		if !ok {
			return fmt.Errorf("directivity is missing %d Hz", band)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < -30 || value > 20 {
			return fmt.Errorf("directivity %d Hz must be between -30 and 20 dB", band)
		}
	}
	return nil
}

func DBToRelativeEnergy(db float64) float64 {
	return math.Pow(10, (db-100)/10)
}

func RelativeEnergyToDB(energy float64) float64 {
	if energy <= 1e-20 {
		return -100
	}
	return 100 + 10*math.Log10(energy)
}

func EnergySubtract(measured, background Spectrum) (Spectrum, []string, error) {
	if err := ValidateSpectrum(measured, "measurement spectrum"); err != nil {
		return nil, nil, err
	}
	if err := ValidateSpectrum(background, "background spectrum"); err != nil {
		return nil, nil, err
	}
	corrected := make(Spectrum, len(constants.OctaveBands))
	unreliable := make([]string, 0)
	for _, band := range constants.OctaveBands {
		key := BandKey(band)
		measuredEnergy := DBToRelativeEnergy(measured[key])
		backgroundEnergy := DBToRelativeEnergy(background[key])
		difference := measuredEnergy - backgroundEnergy
		if measured[key]-background[key] < 3 || difference <= 1e-20 {
			unreliable = append(unreliable, key)
			difference = math.Max(difference, 1e-20)
		}
		corrected[key] = round(RelativeEnergyToDB(difference), 4)
	}
	return corrected, unreliable, nil
}

func OverallDB(s Spectrum) float64 {
	energy := 0.0
	for _, band := range constants.OctaveBands {
		energy += DBToRelativeEnergy(s[BandKey(band)])
	}
	return round(RelativeEnergyToDB(energy), 4)
}

func PropagatedDB(source SourceInput, point PointInput, band int) (float64, error) {
	if source.ReferenceDistanceM <= 0 || source.OperatingFactor <= 0 || source.OperatingFactor > 1 {
		return 0, fmt.Errorf("source %s has invalid reference distance or operating factor", source.SourceCode)
	}
	dx := source.XM - point.XM
	dy := source.YM - point.YM
	dz := source.HeightM - point.HeightM
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	distance = math.Max(distance, source.ReferenceDistanceM)
	key := BandKey(band)
	power, ok := source.Power[key]
	if !ok {
		return 0, fmt.Errorf("source %s is missing %d Hz power", source.SourceCode, band)
	}
	directivity, ok := source.Directivity[key]
	if !ok {
		return 0, fmt.Errorf("source %s is missing %d Hz directivity", source.SourceCode, band)
	}
	spreading := 20 * math.Log10(distance/source.ReferenceDistanceM)
	operatingAdjustment := 10 * math.Log10(source.OperatingFactor)
	return power - spreading + directivity + operatingAdjustment, nil
}

func CanonicalHash(value any) (string, []byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", nil, fmt.Errorf("marshal canonical input: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), payload, nil
}

func SpectrumChecksum(pointID uint, measuredAt string, duration int, spectrum Spectrum) (string, error) {
	canonical := struct {
		PointID    uint     `json:"point_id"`
		MeasuredAt string   `json:"measured_at"`
		Duration   int      `json:"duration"`
		Spectrum   Spectrum `json:"spectrum"`
	}{pointID, measuredAt, duration, spectrum}
	hash, _, err := CanonicalHash(canonical)
	return hash, err
}

func SortedUniqueIDs(values []uint) ([]uint, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one id is required")
	}
	seen := make(map[uint]bool, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			return nil, fmt.Errorf("id must be positive")
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}
