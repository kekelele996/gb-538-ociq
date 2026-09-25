package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"industrial-noise-source-attribution/backend/internal/algorithm"
	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/util"
)

type AttributionRunService struct {
	repository            *repository.AttributionRunRepository
	measurementRepository *repository.NoiseMeasurementRepository
	sourceRepository      *repository.SourceProfileRepository
}

func NewAttributionRunService(runRepo *repository.AttributionRunRepository, measurementRepo *repository.NoiseMeasurementRepository, sourceRepo *repository.SourceProfileRepository) *AttributionRunService {
	return &AttributionRunService{repository: runRepo, measurementRepository: measurementRepo, sourceRepository: sourceRepo}
}

func (s *AttributionRunService) List(ctx context.Context) ([]dto.AttributionRunResponse, error) {
	runs, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.AttributionRunResponse, 0, len(runs))
	for _, run := range runs {
		response, err := attributionResponse(run)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func (s *AttributionRunService) Get(ctx context.Context, id uint) (dto.AttributionRunResponse, error) {
	run, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.AttributionRunResponse{}, mapRepositoryError(err, "归因运行不存在", "归因运行读取冲突")
	}
	return attributionResponse(run)
}

func (s *AttributionRunService) Create(ctx context.Context, request dto.CreateAttributionRunRequest, actor model.Actor, idempotencyKey string) (dto.AttributionRunResponse, bool, error) {
	measurementIDs, err := algorithm.SortedUniqueIDs(request.MeasurementIDs)
	if err != nil {
		return dto.AttributionRunResponse{}, false, util.Validation("测量 ID 无效", err)
	}
	sourceIDs, err := algorithm.SortedUniqueIDs(request.SourceProfileIDs)
	if err != nil {
		return dto.AttributionRunResponse{}, false, util.Validation("声源谱 ID 无效", err)
	}
	measurements, err := s.measurementRepository.GetManyReady(ctx, measurementIDs)
	if err != nil {
		return dto.AttributionRunResponse{}, false, util.Validation("所有测量必须存在并处于 ready 状态", err)
	}
	sources, err := s.sourceRepository.GetManyActive(ctx, sourceIDs)
	if err != nil {
		return dto.AttributionRunResponse{}, false, util.Validation("所有声源谱必须存在并处于 active 状态", err)
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i].ID < measurements[j].ID })
	sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
	measurementInputs, err := buildMeasurementInputs(measurements)
	if err != nil {
		return dto.AttributionRunResponse{}, false, err
	}
	sourceInputs, err := buildSourceInputs(sources)
	if err != nil {
		return dto.AttributionRunResponse{}, false, err
	}
	snapshot := struct {
		AlgorithmVersion string                       `json:"algorithm_version"`
		Measurements     []algorithm.MeasurementInput `json:"measurements"`
		Sources          []algorithm.SourceInput      `json:"sources"`
	}{constants.AlgorithmVersion, measurementInputs, sourceInputs}
	inputHash, snapshotJSON, err := algorithm.CanonicalHash(snapshot)
	if err != nil {
		return dto.AttributionRunResponse{}, false, err
	}
	if existing, findErr := s.repository.FindByInput(ctx, inputHash, constants.AlgorithmVersion); findErr == nil {
		response, responseErr := attributionResponse(existing)
		return response, true, responseErr
	} else if !repository.IsNotFound(findErr) && !strings.Contains(findErr.Error(), gorm.ErrRecordNotFound.Error()) {
		return dto.AttributionRunResponse{}, false, findErr
	}
	result, err := algorithm.FitAttribution(measurementInputs, sourceInputs)
	if err != nil {
		return dto.AttributionRunResponse{}, false, util.Validation("归因算法无法处理当前冻结输入", err)
	}
	finished := time.Now().UTC()
	run := model.AttributionRun{
		RunCode:            "AR-" + strings.ToUpper(inputHash[:10]),
		MeasurementIDsJSON: mustJSON(measurementIDs), SourceProfileIDsJSON: mustJSON(sourceIDs),
		AlgorithmVersion: constants.AlgorithmVersion, InputHash: inputHash,
		InputSnapshotJSON: string(snapshotJSON), NormalizedBandsJSON: mustJSON(result.NormalizedBands),
		ContributionsJSON: mustJSON(result.Contributions), EvidenceJSON: mustJSON(result.Evidence),
		ResidualError: result.ResidualError, AttributionState: string(constants.AttributionCompleted),
		Explanation: result.Explanation, StartedAt: finished.Add(-time.Duration(result.Evidence.ElapsedMillis) * time.Millisecond),
		FinishedAt: &finished, CreatedBy: actor.ID, Version: 1, CreatedAt: finished, UpdatedAt: finished,
	}
	audit := newAudit(actor, "attribution_run.completed", "AttributionRun", 0, map[string]any{"state": constants.AttributionQueued}, run, map[string]any{
		"input_hash": inputHash, "algorithm_version": constants.AlgorithmVersion,
		"idempotency_key": idempotencyKey, "measurement_checksums": measurementChecksums(measurementInputs),
		"matrix_rows": result.Evidence.MatrixRows, "matrix_columns": result.Evidence.MatrixColumns,
		"iterations": result.Evidence.Iterations, "elapsed_millis": result.Evidence.ElapsedMillis,
	})
	if err := s.repository.CreateCalculated(ctx, &run, audit); err != nil {
		if existing, findErr := s.repository.FindByInput(ctx, inputHash, constants.AlgorithmVersion); findErr == nil {
			response, responseErr := attributionResponse(existing)
			return response, true, responseErr
		}
		return dto.AttributionRunResponse{}, false, mapRepositoryError(err, "归因运行不存在", "相同冻结输入正在或已经计算")
	}
	response, err := s.Get(ctx, run.ID)
	return response, false, err
}

func (s *AttributionRunService) Review(ctx context.Context, id uint, request dto.ReviewAttributionRequest, actor model.Actor) (dto.AttributionRunResponse, error) {
	return s.transition(ctx, id, request.Version, constants.AttributionReviewed, request.Note, actor)
}

func (s *AttributionRunService) Confirm(ctx context.Context, id uint, request dto.AttributionActionRequest, actor model.Actor) (dto.AttributionRunResponse, error) {
	run, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.AttributionRunResponse{}, mapRepositoryError(err, "归因运行不存在", "归因运行读取冲突")
	}
	if run.CreatedBy == actor.ID {
		return dto.AttributionRunResponse{}, util.Forbidden("归因运行发起人不得确认自己的结果")
	}
	return s.transitionLoaded(ctx, run, request.Version, constants.AttributionConfirmed, "", actor)
}

func (s *AttributionRunService) Void(ctx context.Context, id uint, request dto.AttributionActionRequest, actor model.Actor) (dto.AttributionRunResponse, error) {
	return s.transition(ctx, id, request.Version, constants.AttributionVoided, "", actor)
}

func (s *AttributionRunService) Compare(ctx context.Context, baseID, otherID uint) (dto.AttributionComparisonResponse, error) {
	base, err := s.Get(ctx, baseID)
	if err != nil {
		return dto.AttributionComparisonResponse{}, err
	}
	other, err := s.Get(ctx, otherID)
	if err != nil {
		return dto.AttributionComparisonResponse{}, err
	}
	baseTop, otherTop := "", ""
	if len(base.Contributions) > 0 {
		baseTop = base.Contributions[0].SourceCode
	}
	if len(other.Contributions) > 0 {
		otherTop = other.Contributions[0].SourceCode
	}
	return dto.AttributionComparisonResponse{
		BaseRunID: baseID, OtherRunID: otherID,
		ResidualDelta:    roundFloat(other.ResidualError-base.ResidualError, 6),
		TopSourceChanged: baseTop != otherTop, BaseTopSource: baseTop, OtherTopSource: otherTop,
		Explanation: "比较仅描述两个冻结输入与算法版本的离线结果差异，不覆盖任何历史记录。",
	}, nil
}

func (s *AttributionRunService) transition(ctx context.Context, id, version uint, to constants.AttributionState, note string, actor model.Actor) (dto.AttributionRunResponse, error) {
	run, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.AttributionRunResponse{}, mapRepositoryError(err, "归因运行不存在", "归因运行读取冲突")
	}
	return s.transitionLoaded(ctx, run, version, to, note, actor)
}

func (s *AttributionRunService) transitionLoaded(ctx context.Context, run model.AttributionRun, version uint, to constants.AttributionState, note string, actor model.Actor) (dto.AttributionRunResponse, error) {
	from := constants.AttributionState(run.AttributionState)
	if !constants.CanTransitionAttribution(from, to) {
		return dto.AttributionRunResponse{}, util.Conflict(fmt.Sprintf("不允许从 %s 迁移到 %s", from, to), nil)
	}
	after := run
	after.AttributionState, after.Version = string(to), version+1
	var reviewer *uint
	if to == constants.AttributionReviewed || to == constants.AttributionConfirmed {
		reviewer = &actor.ID
		after.ReviewedBy = reviewer
	}
	if note != "" {
		after.ReviewNote = note
	}
	audit := newAudit(actor, "attribution_run.state_changed", "AttributionRun", run.ID, run, after, map[string]any{"from": from, "to": to, "expected_version": version})
	if err := s.repository.Transition(ctx, run.ID, version, string(from), string(to), reviewer, note, audit); err != nil {
		return dto.AttributionRunResponse{}, mapRepositoryError(err, "归因运行不存在", "归因运行状态或版本已变化")
	}
	return s.Get(ctx, run.ID)
}

func buildMeasurementInputs(measurements []model.NoiseMeasurement) ([]algorithm.MeasurementInput, error) {
	result := make([]algorithm.MeasurementInput, 0, len(measurements))
	for _, measurement := range measurements {
		bands, err := decodeSpectrum(measurement.OctaveBandsJSON)
		if err != nil {
			return nil, err
		}
		background, err := decodeSpectrum(measurement.MonitoringPoint.BackgroundProfileJSON)
		if err != nil {
			return nil, err
		}
		result = append(result, algorithm.MeasurementInput{
			ID: measurement.ID, Checksum: measurement.SourceChecksum, Bands: bands,
			Point: algorithm.PointInput{ID: measurement.MonitoringPoint.ID, PointCode: measurement.MonitoringPoint.PointCode,
				XM: measurement.MonitoringPoint.XM, YM: measurement.MonitoringPoint.YM,
				HeightM: measurement.MonitoringPoint.HeightM, Background: background},
		})
	}
	return result, nil
}

func buildSourceInputs(sources []model.SourceProfile) ([]algorithm.SourceInput, error) {
	result := make([]algorithm.SourceInput, 0, len(sources))
	for _, source := range sources {
		power, err := decodeSpectrum(source.OctavePowerJSON)
		if err != nil {
			return nil, err
		}
		directivity, err := decodeSpectrum(source.DirectivityJSON)
		if err != nil {
			return nil, err
		}
		result = append(result, algorithm.SourceInput{
			ID: source.ID, SourceCode: source.SourceCode, Name: source.Name,
			XM: source.XM, YM: source.YM, HeightM: source.HeightM,
			ReferenceDistanceM: source.ReferenceDistanceM, Power: power, Directivity: directivity,
			OperatingFactor: source.OperatingFactor, Version: source.Version,
		})
	}
	return result, nil
}

func attributionResponse(run model.AttributionRun) (dto.AttributionRunResponse, error) {
	var measurementIDs, sourceIDs []uint
	var contributions []dto.SourceContribution
	var evidence dto.AttributionEvidence
	var snapshot, normalized any
	fields := []struct {
		raw    string
		target any
	}{
		{run.MeasurementIDsJSON, &measurementIDs}, {run.SourceProfileIDsJSON, &sourceIDs},
		{run.ContributionsJSON, &contributions}, {run.EvidenceJSON, &evidence},
		{run.InputSnapshotJSON, &snapshot}, {run.NormalizedBandsJSON, &normalized},
	}
	for _, field := range fields {
		raw, target := field.raw, field.target
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			return dto.AttributionRunResponse{}, fmt.Errorf("decode stored attribution evidence: %w", err)
		}
	}
	return dto.AttributionRunResponse{
		ID: run.ID, RunCode: run.RunCode, MeasurementIDs: measurementIDs, SourceProfileIDs: sourceIDs,
		AlgorithmVersion: run.AlgorithmVersion, InputHash: run.InputHash,
		InputSnapshot: snapshot, NormalizedBands: normalized, Contributions: contributions,
		Evidence: evidence, ResidualError: run.ResidualError, AttributionState: run.AttributionState,
		Explanation: run.Explanation, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
		CreatedBy: run.CreatedBy, ReviewedBy: run.ReviewedBy, ReviewNote: run.ReviewNote,
		Version: run.Version, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
	}, nil
}

func measurementChecksums(inputs []algorithm.MeasurementInput) []string {
	checksums := make([]string, 0, len(inputs))
	for _, input := range inputs {
		checksums = append(checksums, input.Checksum)
	}
	return checksums
}

func roundFloat(value float64, places int) float64 {
	formatted := fmt.Sprintf("%.*f", places, value)
	var rounded float64
	fmt.Sscanf(formatted, "%f", &rounded)
	return rounded
}
