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

func (s *SourceProfileService) Transition(ctx context.Context, id uint, request dto.ProfileTransitionRequest, actor model.Actor) (dto.ProfileTransitionResponse, error) {
	before, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.ProfileTransitionResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱读取冲突")
	}
	from, to := constants.ProfileState(before.ProfileState), constants.ProfileState(request.ToState)
	if !constants.CanTransitionProfile(from, to) {
		return dto.ProfileTransitionResponse{}, util.Conflict(fmt.Sprintf("不允许从 %s 迁移到 %s", from, to), nil)
	}
	if to == constants.ProfileActive {
		return s.activate(ctx, before, from, request.LockVersion, actor)
	}
	after := before
	after.ProfileState, after.LockVersion = request.ToState, request.LockVersion+1
	audit := newAudit(actor, "source_profile.state_changed", "SourceProfile", id, before, after, map[string]any{"from": from, "to": to, "spectrum_version": before.Version})
	if err := s.repository.Transition(ctx, id, request.LockVersion, string(from), string(to), audit); err != nil {
		return dto.ProfileTransitionResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱状态或版本已变化")
	}
	profile, err := s.Get(ctx, id)
	if err != nil {
		return dto.ProfileTransitionResponse{}, err
	}
	return dto.ProfileTransitionResponse{Profile: profile}, nil
}

// activate 启用目标版本并让同设备当前启用版本自动退出候选。
// 两个版本的条件更新与两条审计记录在同一事务提交，任一步失败都不会留下半套状态；
// 审计元数据记录 replaced / superseded_by，能直接看出谁替代了谁。
func (s *SourceProfileService) activate(ctx context.Context, before model.SourceProfile, from constants.ProfileState, lockVersion uint, actor model.Actor) (dto.ProfileTransitionResponse, error) {
	superseded, findErr := s.repository.FindActiveBySourceCode(ctx, before.SourceCode, before.ID)
	if findErr != nil && !repository.IsNotFound(findErr) {
		return dto.ProfileTransitionResponse{}, findErr
	}
	hasSuperseded := findErr == nil
	after := before
	after.ProfileState, after.LockVersion = string(constants.ProfileActive), lockVersion+1
	metadata := map[string]any{"from": from, "to": constants.ProfileActive, "source_code": before.SourceCode, "spectrum_version": before.Version}
	audits := make([]*model.AuditLog, 0, 2)
	if hasSuperseded {
		supersededAfter := superseded
		supersededAfter.ProfileState, supersededAfter.LockVersion = string(constants.ProfileRetired), superseded.LockVersion+1
		metadata["replaced_profile_id"] = superseded.ID
		metadata["replaced_version"] = superseded.Version
		audits = append(audits, newAudit(actor, "source_profile.superseded", "SourceProfile", superseded.ID, superseded, supersededAfter, map[string]any{
			"from": constants.ProfileActive, "to": constants.ProfileRetired,
			"source_code": superseded.SourceCode, "spectrum_version": superseded.Version,
			"superseded_by_profile_id": before.ID, "superseded_by_version": before.Version,
		}))
	}
	audits = append(audits, newAudit(actor, "source_profile.activated", "SourceProfile", before.ID, before, after, metadata))
	var supersededPtr *model.SourceProfile
	if hasSuperseded {
		supersededPtr = &superseded
	}
	if err := s.repository.Activate(ctx, before.ID, lockVersion, string(from), supersededPtr, audits...); err != nil {
		return dto.ProfileTransitionResponse{}, mapRepositoryError(err, "声源谱不存在", "声源谱状态或版本已变化")
	}
	profile, err := s.Get(ctx, before.ID)
	if err != nil {
		return dto.ProfileTransitionResponse{}, err
	}
	response := dto.ProfileTransitionResponse{Profile: profile}
	if hasSuperseded {
		replaced, err := s.Get(ctx, superseded.ID)
		if err != nil {
			return dto.ProfileTransitionResponse{}, err
		}
		response.Replaced = &replaced
	}
	return response, nil
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
