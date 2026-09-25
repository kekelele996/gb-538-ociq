package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/dto"
	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/util"
)

func newSourceProfileTestService(t *testing.T) (*SourceProfileService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&model.SourceProfile{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	repo := repository.NewSourceProfileRepository(db)
	if err := repo.EnsureSingleActiveIndex(context.Background()); err != nil {
		t.Fatalf("ensure single active index: %v", err)
	}
	return NewSourceProfileService(repo), db
}

func seedProfileRow(t *testing.T, db *gorm.DB, sourceCode string, version uint, state string) model.SourceProfile {
	t.Helper()
	profile := model.SourceProfile{
		SourceCode: sourceCode, Name: "Test source " + sourceCode,
		XM: 1, YM: 2, HeightM: 1.5, ReferenceDistanceM: 1,
		OctavePowerJSON: `{"63":90}`, DirectivityJSON: `{"63":0}`,
		OperatingFactor: 0.8, ProfileState: state, Version: version, LockVersion: 1,
		CreatedBy: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile %s V%d: %v", sourceCode, version, err)
	}
	return profile
}

func testActor() model.Actor {
	return model.Actor{ID: 7, Username: "engineer", DisplayName: "Acoustic Engineer", Role: constants.RoleAcousticEngineer, RequestID: "req-test-1"}
}

func profileStateOf(t *testing.T, db *gorm.DB, id uint) string {
	t.Helper()
	var profile model.SourceProfile
	if err := db.First(&profile, id).Error; err != nil {
		t.Fatalf("reload profile %d: %v", id, err)
	}
	return profile.ProfileState
}

func auditActionsFor(t *testing.T, db *gorm.DB, entityID uint) []model.AuditLog {
	t.Helper()
	var logs []model.AuditLog
	if err := db.Where("entity_type = ? AND entity_id = ?", "SourceProfile", entityID).Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatalf("list audits for %d: %v", entityID, err)
	}
	return logs
}

func TestActivateSupersedesCurrentActiveVersion(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	oldActive := seedProfileRow(t, db, "SRC-A", 1, "active")
	draft := seedProfileRow(t, db, "SRC-A", 2, "draft")

	result, err := service.Transition(context.Background(), draft.ID, dto.ProfileTransitionRequest{ToState: "active", LockVersion: 1}, testActor())
	if err != nil {
		t.Fatalf("activate draft: %v", err)
	}
	if result.Profile.ProfileState != "active" {
		t.Fatalf("expected draft to become active, got %s", result.Profile.ProfileState)
	}
	if result.Replaced == nil || result.Replaced.ID != oldActive.ID || result.Replaced.ProfileState != "retired" {
		t.Fatalf("expected replaced to carry retired old version, got %+v", result.Replaced)
	}
	if got := profileStateOf(t, db, oldActive.ID); got != "retired" {
		t.Fatalf("old version should exit candidates, got %s", got)
	}

	supersededAudits := auditActionsFor(t, db, oldActive.ID)
	if len(supersededAudits) != 1 || supersededAudits[0].Action != "source_profile.superseded" {
		t.Fatalf("expected one superseded audit for old version, got %+v", supersededAudits)
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(supersededAudits[0].MetadataJSON), &metadata); err != nil {
		t.Fatalf("decode superseded metadata: %v", err)
	}
	if metadata["superseded_by_version"] != float64(2) || supersededAudits[0].ActorName != "Acoustic Engineer" {
		t.Fatalf("audit should show who replaced whom, got actor=%s metadata=%v", supersededAudits[0].ActorName, metadata)
	}
	activatedAudits := auditActionsFor(t, db, draft.ID)
	if len(activatedAudits) != 1 || activatedAudits[0].Action != "source_profile.activated" {
		t.Fatalf("expected one activated audit for new version, got %+v", activatedAudits)
	}
	var activatedMetadata map[string]any
	if err := json.Unmarshal([]byte(activatedAudits[0].MetadataJSON), &activatedMetadata); err != nil {
		t.Fatalf("decode activated metadata: %v", err)
	}
	if activatedMetadata["replaced_version"] != float64(1) {
		t.Fatalf("activated audit should record replaced version, got %v", activatedMetadata)
	}
}

func TestSwitchBackToRetiredVersion(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	retired := seedProfileRow(t, db, "SRC-B", 1, "retired")
	active := seedProfileRow(t, db, "SRC-B", 2, "active")

	result, err := service.Transition(context.Background(), retired.ID, dto.ProfileTransitionRequest{ToState: "active", LockVersion: 1}, testActor())
	if err != nil {
		t.Fatalf("switch back to retired version: %v", err)
	}
	if result.Profile.ProfileState != "active" || result.Replaced == nil || result.Replaced.ID != active.ID {
		t.Fatalf("expected retired version re-enabled and current replaced, got %+v", result)
	}
	if got := profileStateOf(t, db, active.ID); got != "retired" {
		t.Fatalf("previously active version should be retired, got %s", got)
	}
}

func TestSecondOperatorSeesConflictAndNoStateChange(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	oldActive := seedProfileRow(t, db, "SRC-C", 1, "active")
	draft := seedProfileRow(t, db, "SRC-C", 2, "draft")
	actor := testActor()

	if _, err := service.Transition(context.Background(), draft.ID, dto.ProfileTransitionRequest{ToState: "active", LockVersion: 1}, actor); err != nil {
		t.Fatalf("first activation should win: %v", err)
	}
	// 第二个人基于过期页面重放同一启用操作：目标已是 active，必须 409 且不再改动任何版本。
	_, err := service.Transition(context.Background(), draft.ID, dto.ProfileTransitionRequest{ToState: "active", LockVersion: 1}, actor)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("second operator must see 409, got %v", err)
	}
	if got := profileStateOf(t, db, draft.ID); got != "active" {
		t.Fatalf("failed retry must not change winner state, got %s", got)
	}
	if got := profileStateOf(t, db, oldActive.ID); got != "retired" {
		t.Fatalf("failed retry must not resurrect old version, got %s", got)
	}
	if count := len(auditActionsFor(t, db, draft.ID)); count != 1 {
		t.Fatalf("failed retry must not write extra audits, got %d", count)
	}
}

func TestActivateRollsBackWhenSiblingLockIsStale(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	oldActive := seedProfileRow(t, db, "SRC-D", 1, "active")
	draft := seedProfileRow(t, db, "SRC-D", 2, "draft")
	repo := repository.NewSourceProfileRepository(db)

	// 模拟并发方已改动旧版本：用陈旧的 lock_version 执行启用，整个事务必须回滚。
	staleSuperseded := oldActive
	staleSuperseded.LockVersion = 99
	audit := newAudit(testActor(), "source_profile.activated", "SourceProfile", draft.ID, draft, draft, map[string]any{})
	err := repo.Activate(context.Background(), draft.ID, 1, "draft", &staleSuperseded, audit)
	if !repository.IsConflict(err) {
		t.Fatalf("stale sibling lock must surface as conflict, got %v", err)
	}
	if got := profileStateOf(t, db, draft.ID); got != "draft" {
		t.Fatalf("rolled-back activation must leave target as draft, got %s", got)
	}
	if got := profileStateOf(t, db, oldActive.ID); got != "active" {
		t.Fatalf("rolled-back activation must leave old version active, got %s", got)
	}
	if count := len(auditActionsFor(t, db, draft.ID)); count != 0 {
		t.Fatalf("rolled-back activation must not persist audits, got %d", count)
	}
	_ = service
}

func TestSecondActiveVersionRejectedByUniqueIndex(t *testing.T) {
	_, db := newSourceProfileTestService(t)
	seedProfileRow(t, db, "SRC-E", 1, "active")
	duplicate := model.SourceProfile{
		SourceCode: "SRC-E", Name: "Duplicate active", XM: 1, YM: 2, HeightM: 1.5, ReferenceDistanceM: 1,
		OctavePowerJSON: `{"63":90}`, DirectivityJSON: `{"63":0}`, OperatingFactor: 0.8,
		ProfileState: "active", Version: 2, LockVersion: 1, CreatedBy: 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	err := db.Create(&duplicate).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("second active version must hit the unique index, got %v", err)
	}
}

func TestRetireKeepsSingleTransitionAudit(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	active := seedProfileRow(t, db, "SRC-F", 1, "active")

	result, err := service.Transition(context.Background(), active.ID, dto.ProfileTransitionRequest{ToState: "retired", LockVersion: 1}, testActor())
	if err != nil {
		t.Fatalf("retire active version: %v", err)
	}
	if result.Profile.ProfileState != "retired" || result.Replaced != nil {
		t.Fatalf("plain retire should not report replaced, got %+v", result)
	}
	logs := auditActionsFor(t, db, active.ID)
	if len(logs) != 1 || logs[0].Action != "source_profile.state_changed" {
		t.Fatalf("expected single state_changed audit, got %+v", logs)
	}
}

func TestRetireDuplicateActivesKeepsHighestVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&model.SourceProfile{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	seedProfileRow(t, db, "SRC-G", 1, "active")
	seedProfileRow(t, db, "SRC-G", 2, "active")
	repo := repository.NewSourceProfileRepository(db)
	if err := repo.RetireDuplicateActives(context.Background()); err != nil {
		t.Fatalf("reconcile duplicates: %v", err)
	}
	if err := repo.EnsureSingleActiveIndex(context.Background()); err != nil {
		t.Fatalf("index must build after reconciliation: %v", err)
	}
	var actives []model.SourceProfile
	if err := db.Where("profile_state = ?", "active").Find(&actives).Error; err != nil {
		t.Fatalf("list actives: %v", err)
	}
	if len(actives) != 1 || actives[0].Version != 2 {
		t.Fatalf("expected only highest version active, got %+v", actives)
	}
}

func TestAttributionRejectsDuplicateDeviceVersions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&model.MonitoringPoint{}, &model.NoiseMeasurement{}, &model.SourceProfile{}, &model.AttributionRun{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	now := time.Now().UTC()
	point := model.MonitoringPoint{
		PointCode: "MP-T", Name: "Test point", XM: 10, YM: 10, HeightM: 1.5, AreaType: "workshop",
		BackgroundProfileJSON: `{"63":40,"125":40,"250":40,"500":40,"1000":40,"2000":40,"4000":40,"8000":40}`,
		OwnerTeam:             "QA", PointState: "active", Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&point).Error; err != nil {
		t.Fatalf("seed point: %v", err)
	}
	measurement := model.NoiseMeasurement{
		MonitoringPointID: point.ID, MeasuredAt: now, DurationS: 900,
		OctaveBandsJSON: `{"63":60,"125":60,"250":60,"500":60,"1000":60,"2000":60,"4000":60,"8000":60}`,
		OverallDBA:      66, BackgroundDBA: 43, SourceChecksum: "checksum-t",
		MeasurementQuality: "valid", MeasurementState: "ready", ImportedBy: 1, Version: 4,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&measurement).Error; err != nil {
		t.Fatalf("seed measurement: %v", err)
	}
	// 模拟历史遗留：同一设备存在两个启用版本（本测试库不建唯一索引）。
	first := seedProfileRow(t, db, "SRC-DUP", 1, "active")
	second := seedProfileRow(t, db, "SRC-DUP", 2, "active")

	service := NewAttributionRunService(
		repository.NewAttributionRunRepository(db),
		repository.NewNoiseMeasurementRepository(db),
		repository.NewSourceProfileRepository(db),
	)
	_, _, err = service.Create(context.Background(), dto.CreateAttributionRunRequest{
		MeasurementIDs: []uint{measurement.ID}, SourceProfileIDs: []uint{first.ID, second.ID},
	}, testActor(), "idem-t")
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 422 {
		t.Fatalf("duplicate device versions must return 422, got %v", err)
	}
}
