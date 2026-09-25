package algorithm

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/mat"
	"industrial-noise-source-attribution/backend/internal/constants"
)

type fitRow struct {
	MeasurementIndex int
	BandIndex        int
	BandHz           int
}

func FitAttribution(measurements []MeasurementInput, sources []SourceInput) (Result, error) {
	started := time.Now()
	if len(measurements) == 0 || len(sources) == 0 {
		return Result{}, fmt.Errorf("measurements and sources are required")
	}

	normalized := make([]Spectrum, 0, len(measurements))
	unreliableSet := make(map[string]bool)
	for _, measurement := range measurements {
		corrected, unreliable, err := EnergySubtract(measurement.Bands, measurement.Point.Background)
		if err != nil {
			return Result{}, fmt.Errorf("measurement %d: %w", measurement.ID, err)
		}
		normalized = append(normalized, corrected)
		for _, band := range unreliable {
			unreliableSet[fmt.Sprintf("measurement:%d:%sHz", measurement.ID, band)] = true
		}
	}
	for _, source := range sources {
		if err := ValidateSpectrum(source.Power, "source power"); err != nil {
			return Result{}, fmt.Errorf("source %d: %w", source.ID, err)
		}
		if err := ValidateDirectivity(source.Directivity); err != nil {
			return Result{}, fmt.Errorf("source %d: %w", source.ID, err)
		}
	}

	rows := len(measurements) * len(constants.OctaveBands)
	cols := len(sources)
	matrixData := make([]float64, rows*cols)
	observations := make([]float64, rows)
	rowMeta := make([]fitRow, rows)
	row := 0
	for measurementIndex, measurement := range measurements {
		for bandIndex, band := range constants.OctaveBands {
			rowMeta[row] = fitRow{measurementIndex, bandIndex, band}
			observations[row] = DBToRelativeEnergy(normalized[measurementIndex][BandKey(band)])
			for sourceIndex, source := range sources {
				propagated, err := PropagatedDB(source, measurement.Point, band)
				if err != nil {
					return Result{}, err
				}
				matrixData[row*cols+sourceIndex] = DBToRelativeEnergy(propagated)
			}
			row++
		}
	}

	a := mat.NewDense(rows, cols, matrixData)
	b := mat.NewVecDense(rows, observations)
	scaledA, columnScales := normalizeColumns(a)
	scaledCoefficients, iterations, converged, objective := projectedGradientNNLS(scaledA, b, 12000, 1e-8)
	coefficients := make([]float64, len(scaledCoefficients))
	for index := range scaledCoefficients {
		coefficients[index] = scaledCoefficients[index] / columnScales[index]
	}
	predicted := make([]float64, rows)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			predicted[i] += a.At(i, j) * coefficients[j]
		}
	}

	residualNumerator := 0.0
	residualDenominator := 0.0
	for i, observed := range observations {
		difference := predicted[i] - observed
		residualNumerator += difference * difference
		residualDenominator += observed * observed
	}
	residual := math.Sqrt(residualNumerator / math.Max(residualDenominator, 1e-20))
	conditionHint := maxColumnCorrelation(a)
	warnings := make([]string, 0)
	if conditionHint >= 0.995 {
		warnings = append(warnings, "候选声源传播谱高度相关，部分贡献不可唯一辨识。")
	}
	if residual > 0.35 {
		warnings = append(warnings, "拟合残差偏高，候选声源可能不完整或测量条件不一致。")
	}
	if len(unreliableSet) > 0 {
		warnings = append(warnings, "部分频带与背景差小于 3 dB，已标记为不可可靠扣除。")
	}

	contributions := buildContributions(a, coefficients, rowMeta, sources)
	unreliable := make([]string, 0, len(unreliableSet))
	for key := range unreliableSet {
		unreliable = append(unreliable, key)
	}
	sort.Strings(unreliable)
	explanation := fmt.Sprintf(
		"使用 %d 次测量的 %d 个频带观测与 %d 个候选声源执行非负最小二乘；相对残差 %.4f。",
		len(measurements), rows, len(sources), residual,
	)
	return Result{
		NormalizedBands: normalized,
		Contributions:   contributions,
		ResidualError:   round(residual, 6),
		Evidence: Evidence{
			MatrixRows: rows, MatrixColumns: cols, Iterations: iterations,
			Converged: converged, ConditionHint: round(conditionHint, 6),
			UnreliableBands: unreliable, Warnings: warnings,
			Objective: round(objective, 10), ElapsedMillis: time.Since(started).Milliseconds(),
		},
		Explanation:   explanation,
		PredictedRows: predicted,
	}, nil
}

func projectedGradientNNLS(a *mat.Dense, b *mat.VecDense, maxIterations int, tolerance float64) ([]float64, int, bool, float64) {
	rows, cols := a.Dims()
	x := make([]float64, cols)
	frobeniusSquared := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			value := a.At(i, j)
			frobeniusSquared += value * value
		}
	}
	step := 1 / math.Max(frobeniusSquared, 1e-20)
	residual := make([]float64, rows)
	for iteration := 1; iteration <= maxIterations; iteration++ {
		for i := 0; i < rows; i++ {
			prediction := 0.0
			for j := 0; j < cols; j++ {
				prediction += a.At(i, j) * x[j]
			}
			residual[i] = prediction - b.AtVec(i)
		}
		maxDelta := 0.0
		maxCoefficient := 1.0
		for j := 0; j < cols; j++ {
			gradient := 0.0
			for i := 0; i < rows; i++ {
				gradient += a.At(i, j) * residual[i]
			}
			next := math.Max(0, x[j]-step*gradient)
			maxDelta = math.Max(maxDelta, math.Abs(next-x[j]))
			x[j] = next
			maxCoefficient = math.Max(maxCoefficient, next)
		}
		if maxDelta/maxCoefficient < tolerance {
			return x, iteration, true, objectiveValue(a, b, x)
		}
	}
	return x, maxIterations, false, objectiveValue(a, b, x)
}

func normalizeColumns(a *mat.Dense) (*mat.Dense, []float64) {
	rows, cols := a.Dims()
	data := make([]float64, rows*cols)
	scales := make([]float64, cols)
	for column := 0; column < cols; column++ {
		normSquared := 0.0
		for row := 0; row < rows; row++ {
			value := a.At(row, column)
			normSquared += value * value
		}
		scales[column] = math.Sqrt(math.Max(normSquared, 1e-30))
		for row := 0; row < rows; row++ {
			data[row*cols+column] = a.At(row, column) / scales[column]
		}
	}
	return mat.NewDense(rows, cols, data), scales
}

func objectiveValue(a *mat.Dense, b *mat.VecDense, x []float64) float64 {
	rows, cols := a.Dims()
	value := 0.0
	for i := 0; i < rows; i++ {
		prediction := 0.0
		for j := 0; j < cols; j++ {
			prediction += a.At(i, j) * x[j]
		}
		difference := prediction - b.AtVec(i)
		value += difference * difference
	}
	return value / 2
}

func maxColumnCorrelation(a *mat.Dense) float64 {
	rows, cols := a.Dims()
	maximum := 0.0
	for left := 0; left < cols; left++ {
		for right := left + 1; right < cols; right++ {
			dot, leftNorm, rightNorm := 0.0, 0.0, 0.0
			for row := 0; row < rows; row++ {
				lv, rv := a.At(row, left), a.At(row, right)
				dot += lv * rv
				leftNorm += lv * lv
				rightNorm += rv * rv
			}
			correlation := dot / math.Sqrt(math.Max(leftNorm*rightNorm, 1e-30))
			maximum = math.Max(maximum, correlation)
		}
	}
	return maximum
}

func buildContributions(a *mat.Dense, coefficients []float64, rows []fitRow, sources []SourceInput) []ContributionResult {
	rowCount, sourceCount := a.Dims()
	sourceEnergy := make([]float64, sourceCount)
	bandEnergy := make([][]float64, sourceCount)
	for sourceIndex := range sources {
		bandEnergy[sourceIndex] = make([]float64, len(constants.OctaveBands))
		for row := 0; row < rowCount; row++ {
			value := a.At(row, sourceIndex) * coefficients[sourceIndex]
			sourceEnergy[sourceIndex] += value
			bandEnergy[sourceIndex][rows[row].BandIndex] += value
		}
	}
	totalEnergy := 0.0
	for _, energy := range sourceEnergy {
		totalEnergy += energy
	}
	result := make([]ContributionResult, 0, sourceCount)
	for sourceIndex, source := range sources {
		bands := make([]BandResult, 0, len(constants.OctaveBands))
		for bandIndex, band := range constants.OctaveBands {
			energy := bandEnergy[sourceIndex][bandIndex]
			bands = append(bands, BandResult{
				BandHz: band, PredictedDB: round(RelativeEnergyToDB(energy), 3),
				EnergyShare: round(100*energy/math.Max(totalEnergy, 1e-20), 3),
			})
		}
		result = append(result, ContributionResult{
			SourceProfileID: source.ID, SourceCode: source.SourceCode, SourceName: source.Name,
			Coefficient:     round(coefficients[sourceIndex], 6),
			ContributionPct: round(100*sourceEnergy[sourceIndex]/math.Max(totalEnergy, 1e-20), 3),
			OverallDB:       round(RelativeEnergyToDB(sourceEnergy[sourceIndex]), 3), Bands: bands,
		})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].ContributionPct > result[j].ContributionPct })
	return result
}
