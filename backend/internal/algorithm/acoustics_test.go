package algorithm

import (
	"math"
	"testing"

	"industrial-noise-source-attribution/backend/internal/constants"
)

func TestEnergySubtractUsesEnergyDomain(t *testing.T) {
	measured := constantSpectrum(70)
	background := constantSpectrum(60)
	corrected, unreliable, err := EnergySubtract(measured, background)
	if err != nil {
		t.Fatalf("EnergySubtract returned error: %v", err)
	}
	if len(unreliable) != 0 {
		t.Fatalf("expected reliable bands, got %v", unreliable)
	}
	want := 10 * math.Log10(math.Pow(10, 7)-math.Pow(10, 6))
	if math.Abs(corrected["1000"]-want) > 0.001 {
		t.Fatalf("energy subtraction = %.4f, want %.4f", corrected["1000"], want)
	}
	if corrected["1000"] == 10 {
		t.Fatal("background was subtracted arithmetically in dB")
	}
}

func TestEnergySubtractMarksSmallMargin(t *testing.T) {
	measured := constantSpectrum(61.5)
	background := constantSpectrum(60)
	_, unreliable, err := EnergySubtract(measured, background)
	if err != nil {
		t.Fatalf("EnergySubtract returned error: %v", err)
	}
	if len(unreliable) != len(constants.OctaveBands) {
		t.Fatalf("unreliable count = %d, want %d", len(unreliable), len(constants.OctaveBands))
	}
}

func TestValidateSpectrumRejectsMissingBand(t *testing.T) {
	spectrum := constantSpectrum(70)
	delete(spectrum, "8000")
	if err := ValidateSpectrum(spectrum, "fixture"); err == nil {
		t.Fatal("expected missing band validation error")
	}
}

func constantSpectrum(value float64) Spectrum {
	result := make(Spectrum, len(constants.OctaveBands))
	for _, band := range constants.OctaveBands {
		result[BandKey(band)] = value
	}
	return result
}
