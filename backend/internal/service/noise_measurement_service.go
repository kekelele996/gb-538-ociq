package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"industrial-noise-source-attribution/backend/internal/algorithm"
	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/util"
)

type NoiseMeasurementService struct {
	repository      *repository.NoiseMeasurementRepository
	pointRepository *repository.MonitoringPointRepository
}

func NewNoiseMeasurementService(repo *repository.NoiseMeasurementRepository, pointRepo *repository.MonitoringPointRepository) *NoiseMeasurementService {
	return &NoiseMeasurementService{repository: repo, pointRepository: pointRepo}
}

func (s *NoiseMeasurementService) List(ctx context.Context) ([]dto.NoiseMeasurementResponse, error) {
	measurements, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.NoiseMeasurementResponse, 0, len(measurements))
	for _, measurement := range measurements {
		response, err := s.toResponse(measurement)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func (s *NoiseMeasurementService) Get(ctx context.Context, id uint) (dto.NoiseMeasurementResponse, error) {
	measurement, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, mapRepositoryError(err, "噪声测量不存在", "噪声测量读取冲突")
	}
	return s.toResponse(measurement)
}

func (s *NoiseMeasurementService) Create(ctx context.Context, request dto.CreateNoiseMeasurementRequest, actor model.Actor) (dto.NoiseMeasurementResponse, bool, error) {
	point, err := s.pointRepository.Get(ctx, request.MonitoringPointID)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, false, mapRepositoryError(err, "监测点不存在", "监测点读取冲突")
	}
	if point.PointState != string(constants.PointActive) {
		return dto.NoiseMeasurementResponse{}, false, util.Conflict("不能向已停用监测点导入测量", nil)
	}
	if err := algorithm.ValidateSpectrum(request.OctaveBands, "measurement spectrum"); err != nil {
		return dto.NoiseMeasurementResponse{}, false, util.Validation("测量必须包含 63-8000 Hz 的全部 8 个频带", err)
	}
	checksum, err := algorithm.SpectrumChecksum(request.MonitoringPointID, request.MeasuredAt.UTC().Format(time.RFC3339Nano), request.DurationS, request.OctaveBands)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, false, err
	}
	if existing, findErr := s.repository.FindByChecksum(ctx, request.MonitoringPointID, checksum); findErr == nil {
		response, responseErr := s.toResponse(existing)
		return response, true, responseErr
	}
	quality, reason := classifyMeasurement(request.OctaveBands, request.OverallDBA, request.BackgroundDBA, request.QualityReason)
	now := time.Now().UTC()
	measurement := model.NoiseMeasurement{
		MonitoringPointID: request.MonitoringPointID, MeasuredAt: request.MeasuredAt.UTC(),
		DurationS: request.DurationS, OctaveBandsJSON: mustJSON(request.OctaveBands),
		OverallDBA: request.OverallDBA, BackgroundDBA: request.BackgroundDBA,
		WeatherNote: request.WeatherNote, SourceChecksum: checksum,
		MeasurementQuality: quality, QualityReason: reason,
		MeasurementState: string(constants.MeasurementCaptured), ImportedBy: actor.ID,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	audit := newAudit(actor, "noise_measurement.imported", "NoiseMeasurement", 0, map[string]any{}, measurement, map[string]any{"checksum": checksum, "band_count": 8})
	if err := s.repository.Create(ctx, &measurement, audit); err != nil {
		return dto.NoiseMeasurementResponse{}, false, mapRepositoryError(err, "噪声测量不存在", "相同测量已导入")
	}
	created, err := s.repository.Get(ctx, measurement.ID)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, false, err
	}
	response, err := s.toResponse(created)
	return response, false, err
}

func (s *NoiseMeasurementService) Transition(ctx context.Context, id uint, request dto.MeasurementTransitionRequest, actor model.Actor) (dto.NoiseMeasurementResponse, error) {
	before, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, mapRepositoryError(err, "噪声测量不存在", "噪声测量读取冲突")
	}
	if actor.Role == constants.RoleDataAnalyst && before.ImportedBy != actor.ID {
		return dto.NoiseMeasurementResponse{}, util.Forbidden("数据分析员只能推进本人导入的测量")
	}
	from := constants.MeasurementState(before.MeasurementState)
	to := constants.MeasurementState(request.ToState)
	if !constants.CanTransitionMeasurement(from, to) {
		return dto.NoiseMeasurementResponse{}, util.Conflict(fmt.Sprintf("不允许从 %s 迁移到 %s", from, to), nil)
	}
	if to == constants.MeasurementValidated && before.MeasurementQuality != string(constants.QualityValid) {
		return dto.NoiseMeasurementResponse{}, util.Validation("非 valid 测量不能进入 validated，请拒绝或重新导入", nil)
	}
	quality, reason := "", ""
	if to == constants.MeasurementNormalized {
		bands, decodeErr := decodeSpectrum(before.OctaveBandsJSON)
		if decodeErr != nil {
			return dto.NoiseMeasurementResponse{}, decodeErr
		}
		background, decodeErr := decodeSpectrum(before.MonitoringPoint.BackgroundProfileJSON)
		if decodeErr != nil {
			return dto.NoiseMeasurementResponse{}, decodeErr
		}
		_, unreliable, subtractErr := algorithm.EnergySubtract(bands, background)
		if subtractErr != nil {
			return dto.NoiseMeasurementResponse{}, util.Validation("背景扣除失败", subtractErr)
		}
		if len(unreliable) > 4 {
			return dto.NoiseMeasurementResponse{}, util.Validation("超过一半频带无法可靠扣除背景，测量不能归一化", nil)
		}
		if len(unreliable) > 0 {
			quality = string(constants.QualityContaminated)
			reason = fmt.Sprintf("%d 个频带与背景差小于 3 dB，已保留不可可靠标记", len(unreliable))
		}
	}
	after := before
	after.MeasurementState = request.ToState
	after.Version = request.Version + 1
	if quality != "" {
		after.MeasurementQuality, after.QualityReason = quality, reason
	}
	audit := newAudit(actor, "noise_measurement.state_changed", "NoiseMeasurement", id, before, after, map[string]any{"from": from, "to": to, "expected_version": request.Version})
	if err := s.repository.Transition(ctx, id, request.Version, string(from), string(to), quality, reason, audit); err != nil {
		return dto.NoiseMeasurementResponse{}, mapRepositoryError(err, "噪声测量不存在", "测量状态或版本已变化")
	}
	return s.Get(ctx, id)
}

func classifyMeasurement(bands map[string]float64, overall, background float64, suppliedReason string) (string, string) {
	for _, value := range bands {
		if value >= 130 || math.IsInf(value, 0) || math.IsNaN(value) {
			return string(constants.QualityClipped), "存在削顶或无效频带"
		}
	}
	if overall-background < 3 {
		return string(constants.QualityContaminated), "总声级与背景差小于 3 dB"
	}
	if suppliedReason != "" {
		return string(constants.QualityValid), suppliedReason
	}
	return string(constants.QualityValid), "频带完整，动态范围与背景差满足离线拟合条件"
}

func (s *NoiseMeasurementService) toResponse(measurement model.NoiseMeasurement) (dto.NoiseMeasurementResponse, error) {
	bands, err := decodeSpectrum(measurement.OctaveBandsJSON)
	if err != nil {
		return dto.NoiseMeasurementResponse{}, err
	}
	response := dto.NoiseMeasurementResponse{
		ID: measurement.ID, MonitoringPointID: measurement.MonitoringPointID,
		PointCode: measurement.MonitoringPoint.PointCode, PointName: measurement.MonitoringPoint.Name,
		MeasuredAt: measurement.MeasuredAt, DurationS: measurement.DurationS, OctaveBands: bands,
		OverallDBA: measurement.OverallDBA, BackgroundDBA: measurement.BackgroundDBA,
		WeatherNote: measurement.WeatherNote, SourceChecksum: measurement.SourceChecksum,
		MeasurementQuality: measurement.MeasurementQuality, QualityReason: measurement.QualityReason,
		MeasurementState: measurement.MeasurementState, ImportedBy: measurement.ImportedBy,
		Version: measurement.Version, CreatedAt: measurement.CreatedAt, UpdatedAt: measurement.UpdatedAt,
	}
	if measurement.MeasurementState == string(constants.MeasurementNormalized) || measurement.MeasurementState == string(constants.MeasurementReady) {
		background, decodeErr := decodeSpectrum(measurement.MonitoringPoint.BackgroundProfileJSON)
		if decodeErr != nil {
			return dto.NoiseMeasurementResponse{}, decodeErr
		}
		normalized, _, subtractErr := algorithm.EnergySubtract(bands, background)
		if subtractErr != nil {
			return dto.NoiseMeasurementResponse{}, subtractErr
		}
		response.NormalizedBands = normalized
	}
	return response, nil
}
