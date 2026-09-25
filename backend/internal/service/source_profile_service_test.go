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
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.SourceProfile{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return NewSourceProfileService(repository.NewSourceProfileRepository(db)), db
}

func seedTestProfile(t *testing.T, db *gorm.DB, code string, version uint, state string) model.SourceProfile {
	t.Helper()
	now := time.Now().UTC()
	profile := model.SourceProfile{
		SourceCode: code, Name: code + " device", XM: 1, YM: 2, HeightM: 1.5, ReferenceDistanceM: 1,
		OctavePowerJSON: `{"63":90,"125":91,"250":92,"500":93,"1000":94,"2000":95,"4000":96,"8000":97}`,
		DirectivityJSON: `{"63":0,"125":0,"250":0,"500":0,"1000":0,"2000":0,"4000":0,"8000":0}`,
		OperatingFactor: 0.8, ProfileState: state, Version: version, LockVersion: 1,
		CreatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed source profile: %v", err)
	}
	return profile
}

func testActor(requestID string) model.Actor {
	return model.Actor{ID: 7, Username: "engineer", DisplayName: "Acoustic Engineer", Role: constants.RoleAcousticEngineer, RequestID: requestID}
}

func reloadProfile(t *testing.T, db *gorm.DB, id uint) model.SourceProfile {
	t.Helper()
	var profile model.SourceProfile
	if err := db.First(&profile, id).Error; err != nil {
		t.Fatalf("reload source profile %d: %v", id, err)
	}
	return profile
}

func listAudits(t *testing.T, db *gorm.DB) []model.AuditLog {
	t.Helper()
	var audits []model.AuditLog
	if err := db.Order("id ASC").Find(&audits).Error; err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	return audits
}

func expectConflict(t *testing.T, err error) {
	t.Helper()
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("expected HTTP 409 conflict, got %v", err)
	}
}

func TestActivateSupersedesCurrentActiveAtomically(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-A", 1, "active")
	v2 := seedTestProfile(t, db, "SRC-A", 2, "draft")

	response, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v1.ID,
	}, testActor("req-activate-1"))
	if err != nil {
		t.Fatalf("activate draft: %v", err)
	}
	if response.ProfileState != "active" || response.LockVersion != 2 || response.SupersededByID != nil {
		t.Fatalf("unexpected activated version: %+v", response)
	}
	old := reloadProfile(t, db, v1.ID)
	if old.ProfileState != "retired" || old.LockVersion != 2 || old.SupersededByID == nil || *old.SupersededByID != v2.ID {
		t.Fatalf("old version must exit candidates pointing at its replacement: %+v", old)
	}

	audits := listAudits(t, db)
	if len(audits) != 2 {
		t.Fatalf("activation with supersede must write two audit rows, got %d", len(audits))
	}
	byAction := map[string]model.AuditLog{}
	for _, audit := range audits {
		if audit.RequestID != "req-activate-1" || audit.ActorName != "Acoustic Engineer" {
			t.Fatalf("audit must carry shared request ID and actor: %+v", audit)
		}
		byAction[audit.Action] = audit
	}
	activated, superseded := byAction["source_profile.activated"], byAction["source_profile.superseded"]
	if activated.EntityID != v2.ID || superseded.EntityID != v1.ID {
		t.Fatalf("audit rows must name who replaced whom: %+v / %+v", activated, superseded)
	}
	var activatedMeta, supersededMeta map[string]any
	if err := json.Unmarshal([]byte(activated.MetadataJSON), &activatedMeta); err != nil {
		t.Fatalf("decode activated metadata: %v", err)
	}
	if err := json.Unmarshal([]byte(superseded.MetadataJSON), &supersededMeta); err != nil {
		t.Fatalf("decode superseded metadata: %v", err)
	}
	if activatedMeta["supersedes_version"] != float64(1) || supersededMeta["superseded_by_version"] != float64(2) {
		t.Fatalf("metadata must record the replacement chain: %v / %v", activatedMeta, supersededMeta)
	}
}

func TestSwitchBackToRetiredVersion(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-B", 1, "active")
	v2 := seedTestProfile(t, db, "SRC-B", 2, "draft")
	if _, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v1.ID,
	}, testActor("req-switch-1")); err != nil {
		t.Fatalf("activate v2: %v", err)
	}

	old := reloadProfile(t, db, v1.ID)
	response, err := service.Transition(context.Background(), v1.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: old.LockVersion, ExpectedActiveID: &v2.ID,
	}, testActor("req-switch-2"))
	if err != nil {
		t.Fatalf("switch back to retired version: %v", err)
	}
	if response.ProfileState != "active" || response.SupersededByID != nil {
		t.Fatalf("switched-back version must be active without replacement pointer: %+v", response)
	}
	current := reloadProfile(t, db, v2.ID)
	if current.ProfileState != "retired" || current.SupersededByID == nil || *current.SupersededByID != v1.ID {
		t.Fatalf("previously active version must retire pointing at v1: %+v", current)
	}
}

func TestSecondOperatorWithStaleViewConflictsCleanly(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-C", 1, "active")
	v2 := seedTestProfile(t, db, "SRC-C", 2, "draft")
	v3 := seedTestProfile(t, db, "SRC-C", 3, "draft")

	if _, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v1.ID,
	}, testActor("req-first")); err != nil {
		t.Fatalf("first operator must succeed: %v", err)
	}

	_, err := service.Transition(context.Background(), v3.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v1.ID,
	}, testActor("req-second"))
	expectConflict(t, err)

	if got := reloadProfile(t, db, v3.ID); got.ProfileState != "draft" || got.LockVersion != 1 {
		t.Fatalf("losing version must keep its pre-conflict state: %+v", got)
	}
	if got := reloadProfile(t, db, v2.ID); got.ProfileState != "active" || got.LockVersion != 2 {
		t.Fatalf("winning version must stay active: %+v", got)
	}
	if got := reloadProfile(t, db, v1.ID); got.ProfileState != "retired" || got.SupersededByID == nil || *got.SupersededByID != v2.ID {
		t.Fatalf("superseded version must stay retired: %+v", got)
	}
	if audits := listAudits(t, db); len(audits) != 2 {
		t.Fatalf("failed operation must not write audit rows, got %d", len(audits))
	}
}

func TestStaleLockVersionConflictsInsideTransaction(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-D", 1, "active")
	v2 := seedTestProfile(t, db, "SRC-D", 2, "draft")
	if _, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v1.ID,
	}, testActor("req-lock-1")); err != nil {
		t.Fatalf("activate v2: %v", err)
	}

	// 第二个操作者刷新列表后基于过期的 lock_version 重复启用同一版本。
	_, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: &v2.ID,
	}, testActor("req-lock-2"))
	expectConflict(t, err)
	if audits := listAudits(t, db); len(audits) != 2 {
		t.Fatalf("rejected retry must not append audit rows, got %d", len(audits))
	}
}

func TestActivateWithoutCurrentActive(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-E", 1, "draft")

	response, err := service.Transition(context.Background(), v1.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: nil,
	}, testActor("req-first-active"))
	if err != nil {
		t.Fatalf("activate the only draft: %v", err)
	}
	if response.ProfileState != "active" {
		t.Fatalf("expected active state, got %+v", response)
	}
	audits := listAudits(t, db)
	if len(audits) != 1 || audits[0].Action != "source_profile.activated" {
		t.Fatalf("single activation must write exactly one audit row: %+v", audits)
	}
}

func TestActivateExpectingNoneWhileSiblingActiveConflicts(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-F", 1, "active")
	v2 := seedTestProfile(t, db, "SRC-F", 2, "draft")

	_, err := service.Transition(context.Background(), v2.ID, dto.ProfileTransitionRequest{
		ToState: "active", LockVersion: 1, ExpectedActiveID: nil,
	}, testActor("req-expect-none"))
	expectConflict(t, err)
	if got := reloadProfile(t, db, v2.ID); got.ProfileState != "draft" {
		t.Fatalf("rejected activation must leave the draft untouched: %+v", got)
	}
	if got := reloadProfile(t, db, v1.ID); got.ProfileState != "active" || got.SupersededByID != nil {
		t.Fatalf("active version must remain untouched: %+v", got)
	}
	if audits := listAudits(t, db); len(audits) != 0 {
		t.Fatalf("no audit rows may survive a rejected activation, got %d", len(audits))
	}
}

func TestManualRetireKeepsLineageAndClearsReplacement(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-G", 1, "active")

	response, err := service.Transition(context.Background(), v1.ID, dto.ProfileTransitionRequest{
		ToState: "retired", LockVersion: 1,
	}, testActor("req-retire"))
	if err != nil {
		t.Fatalf("manual retire: %v", err)
	}
	if response.ProfileState != "retired" || response.SupersededByID != nil {
		t.Fatalf("manual retire must not invent a replacement: %+v", response)
	}
	audits := listAudits(t, db)
	if len(audits) != 1 || audits[0].Action != "source_profile.state_changed" {
		t.Fatalf("manual retire keeps the state_changed audit action: %+v", audits)
	}
}

func TestIllegalProfileTransitionRejected(t *testing.T) {
	service, db := newSourceProfileTestService(t)
	v1 := seedTestProfile(t, db, "SRC-H", 1, "draft")

	_, err := service.Transition(context.Background(), v1.ID, dto.ProfileTransitionRequest{
		ToState: "retired", LockVersion: 1,
	}, testActor("req-illegal"))
	expectConflict(t, err)
	if got := reloadProfile(t, db, v1.ID); got.ProfileState != "draft" {
		t.Fatalf("illegal transition must not change state: %+v", got)
	}
}
