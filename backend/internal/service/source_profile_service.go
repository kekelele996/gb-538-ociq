package service

import (
	"context"
	"fmt"
	"time"

	"industrial-noise-source-attribution/backend/internal/algorithm"
	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/util"
)

type SourceProfileService struct {
	repository *repository.SourceProfileRepository
}

func NewSourceProfileService(repo *repository.SourceProfileRepository) *SourceProfileService {
	return &SourceProfileService{repository: repo}
}

func (s *SourceProfileService) List(ctx context.Context) ([]dto.SourceProfileResponse, error) {
	profiles, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.SourceProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		response, err := sourceProfileResponse(profile)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func (s *SourceProfileService) Get(ctx context.Context, id uint) (dto.SourceProfileResponse, error) {
	profile, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.SourceProfileResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱读取冲突")
	}
	return sourceProfileResponse(profile)
}

func (s *SourceProfileService) Create(ctx context.Context, request dto.CreateSourceProfileRequest, actor model.Actor) (dto.SourceProfileResponse, error) {
	if err := algorithm.ValidateSpectrum(request.OctavePower, "source power"); err != nil {
		return dto.SourceProfileResponse{}, util.Validation("声源功率谱必须包含完整有效的 8 个频带", err)
	}
	if err := algorithm.ValidateDirectivity(request.Directivity); err != nil {
		return dto.SourceProfileResponse{}, util.Validation("方向性修正必须包含 8 个频带且范围为 -30 至 20 dB", err)
	}
	version, err := s.repository.NextVersion(ctx, request.SourceCode)
	if err != nil {
		return dto.SourceProfileResponse{}, err
	}
	now := time.Now().UTC()
	profile := model.SourceProfile{
		SourceCode: request.SourceCode, Name: request.Name, XM: request.XM, YM: request.YM,
		HeightM: request.HeightM, ReferenceDistanceM: request.ReferenceDistanceM,
		OctavePowerJSON: mustJSON(request.OctavePower), DirectivityJSON: mustJSON(request.Directivity),
		OperatingFactor: request.OperatingFactor, ProfileState: string(constants.ProfileDraft),
		Version: version, LockVersion: 1, CreatedBy: actor.ID, CreatedAt: now, UpdatedAt: now,
	}
	audit := newAudit(actor, "source_profile.created", "SourceProfile", 0, map[string]any{}, profile, map[string]any{"source_code": profile.SourceCode, "spectrum_version": version})
	if err := s.repository.Create(ctx, &profile, audit); err != nil {
		return dto.SourceProfileResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱版本已存在")
	}
	return sourceProfileResponse(profile)
}

func (s *SourceProfileService) Transition(ctx context.Context, id uint, request dto.ProfileTransitionRequest, actor model.Actor) (dto.SourceProfileResponse, error) {
	before, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.SourceProfileResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱读取冲突")
	}
	from, to := constants.ProfileState(before.ProfileState), constants.ProfileState(request.ToState)
	if !constants.CanTransitionProfile(from, to) {
		return dto.SourceProfileResponse{}, util.Conflict(fmt.Sprintf("不允许从 %s 迁移到 %s", from, to), nil)
	}
	after := before
	after.ProfileState, after.LockVersion = request.ToState, request.LockVersion+1
	audit := newAudit(actor, "source_profile.state_changed", "SourceProfile", id, before, after, map[string]any{"from": from, "to": to, "spectrum_version": before.Version})
	if err := s.repository.Transition(ctx, id, request.LockVersion, string(from), string(to), audit); err != nil {
		return dto.SourceProfileResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱状态或版本已变化")
	}
	return s.Get(ctx, id)
}

func sourceProfileResponse(profile model.SourceProfile) (dto.SourceProfileResponse, error) {
	power, err := decodeSpectrum(profile.OctavePowerJSON)
	if err != nil {
		return dto.SourceProfileResponse{}, err
	}
	directivity, err := decodeSpectrum(profile.DirectivityJSON)
	if err != nil {
		return dto.SourceProfileResponse{}, err
	}
	return dto.SourceProfileResponse{
		ID: profile.ID, SourceCode: profile.SourceCode, Name: profile.Name,
		XM: profile.XM, YM: profile.YM, HeightM: profile.HeightM,
		ReferenceDistanceM: profile.ReferenceDistanceM, OctavePower: power, Directivity: directivity,
		OperatingFactor: profile.OperatingFactor, ProfileState: profile.ProfileState,
		Version: profile.Version, LockVersion: profile.LockVersion, CreatedBy: profile.CreatedBy,
		CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt,
	}, nil
}
