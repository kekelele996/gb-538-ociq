package algorithm

import (
	"math"
	"testing"

	"industrial-noise-source-attribution/backend/internal/constants"
)

func TestFitAttributionRecoversSyntheticContributions(t *testing.T) {
	background := constantSpectrum(35)
	points := []PointInput{
		{ID: 1, PointCode: "P1", XM: 20, YM: 0, HeightM: 1.5, Background: background},
		{ID: 2, PointCode: "P2", XM: 0, YM: 30, HeightM: 1.5, Background: background},
		{ID: 3, PointCode: "P3", XM: -25, YM: 12, HeightM: 1.5, Background: background},
	}
	sources := []SourceInput{
		{ID: 1, SourceCode: "S1", Name: "Source 1", XM: 0, YM: 0, HeightM: 1, ReferenceDistanceM: 1, Power: varyingSpectrum(94, .8), Directivity: constantSpectrum(0), OperatingFactor: .9, Version: 1},
		{ID: 2, SourceCode: "S2", Name: "Source 2", XM: 35, YM: 18, HeightM: 4, ReferenceDistanceM: 1, Power: varyingSpectrum(88, -.45), Directivity: constantSpectrum(-1), OperatingFactor: .7, Version: 2},
	}
	wantCoefficients := []float64{1.7, .45}
	measurements := make([]MeasurementInput, 0, len(points))
	for index, point := range points {
		bands := make(Spectrum, len(constants.OctaveBands))
		for _, band := range constants.OctaveBands {
			energy := DBToRelativeEnergy(background[BandKey(band)])
			for sourceIndex, source := range sources {
				propagated, err := PropagatedDB(source, point, band)
				if err != nil {
					t.Fatalf("PropagatedDB returned error: %v", err)
				}
				energy += wantCoefficients[sourceIndex] * DBToRelativeEnergy(propagated)
			}
			bands[BandKey(band)] = RelativeEnergyToDB(energy)
		}
		measurements = append(measurements, MeasurementInput{ID: uint(index + 1), Checksum: "fixture", Point: point, Bands: bands})
	}
	result, err := FitAttribution(measurements, sources)
	if err != nil {
		t.Fatalf("FitAttribution returned error: %v", err)
	}
	if !result.Evidence.Converged || result.ResidualError > 0.001 {
		t.Fatalf("fit did not converge accurately: evidence=%+v contributions=%+v", result.Evidence, result.Contributions)
	}
	coefficients := map[uint]float64{}
	for _, contribution := range result.Contributions {
		coefficients[contribution.SourceProfileID] = contribution.Coefficient
	}
	for index, want := range wantCoefficients {
		if math.Abs(coefficients[uint(index+1)]-want) > 0.03 {
			t.Fatalf("source %d coefficient = %.5f, want %.5f", index+1, coefficients[uint(index+1)], want)
		}
	}
}

func varyingSpectrum(start, step float64) Spectrum {
	result := make(Spectrum, len(constants.OctaveBands))
	for index, band := range constants.OctaveBands {
		result[BandKey(band)] = start + float64(index)*step
	}
	return result
}
