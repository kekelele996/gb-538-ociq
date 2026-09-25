package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"industrial-noise-source-attribution/backend/internal/algorithm"
	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/model"
)

type Config struct {
	Port                      string
	DBDriver                  string
	DBDSN                     string
	JWTSecret                 string
	JWTExpiry                 time.Duration
	LoginLimitPerMinute       int
	ImportLimitPerMinute      int
	AttributionLimitPerMinute int
	ShutdownTimeout           time.Duration
}

func Load() (Config, error) {
	config := Config{
		Port: env("PORT", "8080"), DBDriver: env("DB_DRIVER", "postgres"),
		DBDSN:     env("DB_DSN", "host=localhost user=noisetrace password=noisetrace_pwd dbname=noisetrace port=5432 sslmode=disable"),
		JWTSecret: env("JWT_SECRET", "development-only-change-me"), JWTExpiry: durationEnv("JWT_EXPIRY", 8*time.Hour),
		LoginLimitPerMinute:       intEnv("LOGIN_LIMIT_PER_MINUTE", 20),
		ImportLimitPerMinute:      intEnv("IMPORT_LIMIT_PER_MINUTE", 30),
		AttributionLimitPerMinute: intEnv("ATTRIBUTION_LIMIT_PER_MINUTE", 30),
		ShutdownTimeout:           durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if config.Port == "" {
		return Config{}, fmt.Errorf("PORT must not be empty")
	}
	if config.DBDriver != "postgres" && config.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", config.DBDriver)
	}
	if config.DBDSN == "" {
		return Config{}, fmt.Errorf("DB_DSN must not be empty")
	}
	if len(config.JWTSecret) < 16 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 16 characters")
	}
	return config, nil
}

func OpenDatabase(config Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if config.DBDriver == "sqlite" {
		dialector = sqlite.Open(config.DBDSN)
	} else {
		dialector = postgres.Open(config.DBDSN)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn), TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", config.DBDriver, err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.MonitoringPoint{}, &model.NoiseMeasurement{}, &model.SourceProfile{},
		&model.AttributionRun{}, &model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	if err := seedDatabase(db); err != nil {
		return nil, fmt.Errorf("seed database: %w", err)
	}
	return db, nil
}

func CloseDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql database: %w", err)
	}
	return sqlDB.Close()
}

func seedDatabase(db *gorm.DB) error {
	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		users, err := seedUsers()
		if err != nil {
			return err
		}
		if err := tx.Create(&users).Error; err != nil {
			return err
		}
		userID := make(map[string]uint, len(users))
		for _, user := range users {
			userID[user.Username] = user.ID
		}
		points := seedPoints()
		if err := tx.Create(&points).Error; err != nil {
			return err
		}
		for _, point := range points {
			if err := tx.Create(seedAudit(userID["admin"], "System Administrator", "monitoring_point.seeded", "MonitoringPoint", point.ID, point, map[string]any{"source": "deterministic_fixture"})).Error; err != nil {
				return err
			}
		}
		profiles := seedProfiles(userID["engineer"])
		if err := tx.Create(&profiles).Error; err != nil {
			return err
		}
		for _, profile := range profiles {
			if err := tx.Create(seedAudit(userID["engineer"], "Acoustic Engineer", "source_profile.seeded", "SourceProfile", profile.ID, profile, map[string]any{"spectrum_version": profile.Version})).Error; err != nil {
				return err
			}
		}
		measurements, err := seedMeasurements(points, userID["analyst"])
		if err != nil {
			return err
		}
		if err := tx.Create(&measurements).Error; err != nil {
			return err
		}
		for _, measurement := range measurements {
			if err := tx.Create(seedAudit(userID["analyst"], "Noise Data Analyst", "noise_measurement.seeded", "NoiseMeasurement", measurement.ID, measurement, map[string]any{"checksum": measurement.SourceChecksum})).Error; err != nil {
				return err
			}
		}
		return seedAttribution(tx, points, measurements, profiles, userID["analyst"])
	})
}

func seedUsers() ([]model.User, error) {
	definitions := []struct{ username, password, display, role string }{
		{"admin", "admin123", "System Administrator", constants.RoleAdmin},
		{"engineer", "engineer123", "Acoustic Engineer", constants.RoleAcousticEngineer},
		{"analyst", "analyst123", "Noise Data Analyst", constants.RoleDataAnalyst},
		{"reviewer", "reviewer123", "Independent Reviewer", constants.RoleReviewer},
		{"auditor", "auditor123", "Compliance Auditor", constants.RoleAuditor},
	}
	users := make([]model.User, 0, len(definitions))
	for _, definition := range definitions {
		hash, err := bcrypt.GenerateFromPassword([]byte(definition.password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash seed password: %w", err)
		}
		users = append(users, model.User{
			Username: definition.username, PasswordHash: string(hash), DisplayName: definition.display,
			Role: definition.role, Active: true, CreatedAt: time.Now().UTC(),
		})
	}
	return users, nil
}

func seedPoints() []model.MonitoringPoint {
	backgrounds := []algorithm.Spectrum{
		{"63": 51.2, "125": 48.6, "250": 45.1, "500": 42.8, "1000": 40.4, "2000": 38.3, "4000": 36.2, "8000": 33.4},
		{"63": 49.4, "125": 46.9, "250": 43.7, "500": 40.9, "1000": 38.1, "2000": 36.0, "4000": 34.3, "8000": 31.8},
		{"63": 47.8, "125": 44.4, "250": 41.2, "500": 38.7, "1000": 36.5, "2000": 34.7, "4000": 32.1, "8000": 29.6},
	}
	now := time.Now().UTC().Add(-72 * time.Hour)
	return []model.MonitoringPoint{
		{PointCode: "MP-EAST-01", Name: "East boundary receptor", XM: 85, YM: 18, HeightM: 1.5, AreaType: "boundary", BackgroundProfileJSON: mustSeedJSON(backgrounds[0]), OwnerTeam: "Occupational Hygiene", PointState: "active", Version: 1, CreatedAt: now, UpdatedAt: now},
		{PointCode: "MP-NORTH-02", Name: "North workshop aisle", XM: 32, YM: 74, HeightM: 1.5, AreaType: "workshop", BackgroundProfileJSON: mustSeedJSON(backgrounds[1]), OwnerTeam: "Acoustics Lab", PointState: "active", Version: 1, CreatedAt: now, UpdatedAt: now},
		{PointCode: "MP-OFFICE-03", Name: "Control room facade", XM: -28, YM: 41, HeightM: 1.2, AreaType: "office", BackgroundProfileJSON: mustSeedJSON(backgrounds[2]), OwnerTeam: "EHS Review", PointState: "active", Version: 1, CreatedAt: now, UpdatedAt: now},
	}
}

func seedProfiles(createdBy uint) []model.SourceProfile {
	definitions := []struct {
		code, name string
		x, y, h    float64
		factor     float64
		power      algorithm.Spectrum
		direction  algorithm.Spectrum
	}{
		{"SRC-COMP-01", "Compressor train A", 0, 0, 1.8, .92, algorithm.Spectrum{"63": 94, "125": 98, "250": 101, "500": 99, "1000": 96, "2000": 92, "4000": 88, "8000": 83}, flatDirection(0)},
		{"SRC-FAN-02", "Cooling tower fan 2", 42, 8, 6.5, .78, algorithm.Spectrum{"63": 91, "125": 96, "250": 94, "500": 91, "1000": 87, "2000": 83, "4000": 79, "8000": 75}, flatDirection(-1.5)},
		{"SRC-PUMP-03", "Transfer pump P-203", -12, 38, 1.1, .65, algorithm.Spectrum{"63": 82, "125": 86, "250": 91, "500": 94, "1000": 92, "2000": 88, "4000": 82, "8000": 77}, flatDirection(1)},
	}
	now := time.Now().UTC().Add(-48 * time.Hour)
	profiles := make([]model.SourceProfile, 0, len(definitions))
	for _, definition := range definitions {
		profiles = append(profiles, model.SourceProfile{
			SourceCode: definition.code, Name: definition.name, XM: definition.x, YM: definition.y, HeightM: definition.h,
			ReferenceDistanceM: 1, OctavePowerJSON: mustSeedJSON(definition.power), DirectivityJSON: mustSeedJSON(definition.direction),
			OperatingFactor: definition.factor, ProfileState: "active", Version: 1, LockVersion: 1,
			CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
		})
	}
	return profiles
}

func seedMeasurements(points []model.MonitoringPoint, importedBy uint) ([]model.NoiseMeasurement, error) {
	spectra := []algorithm.Spectrum{
		{"63": 65.8, "125": 69.4, "250": 71.6, "500": 68.8, "1000": 64.9, "2000": 60.7, "4000": 56.2, "8000": 50.8},
		{"63": 67.1, "125": 71.2, "250": 69.7, "500": 66.3, "1000": 62.1, "2000": 57.6, "4000": 53.3, "8000": 48.5},
		{"63": 58.9, "125": 61.5, "250": 65.4, "500": 67.8, "1000": 65.2, "2000": 60.4, "4000": 54.8, "8000": 49.1},
	}
	now := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Minute)
	measurements := make([]model.NoiseMeasurement, 0, len(points))
	for index, point := range points {
		measuredAt := now.Add(time.Duration(index) * 15 * time.Minute)
		checksum, err := algorithm.SpectrumChecksum(point.ID, measuredAt.Format(time.RFC3339Nano), 900, spectra[index])
		if err != nil {
			return nil, err
		}
		measurements = append(measurements, model.NoiseMeasurement{
			MonitoringPointID: point.ID, MeasuredAt: measuredAt, DurationS: 900,
			OctaveBandsJSON: mustSeedJSON(spectra[index]), OverallDBA: algorithm.OverallDB(spectra[index]),
			BackgroundDBA: 43.2 - float64(index), WeatherNote: "Dry, wind below 2 m/s; offline fixture",
			SourceChecksum: checksum, MeasurementQuality: "valid", QualityReason: "Complete octave bands and adequate background margin",
			MeasurementState: "ready", ImportedBy: importedBy, Version: 4, CreatedAt: now, UpdatedAt: now,
		})
	}
	return measurements, nil
}

func seedAttribution(tx *gorm.DB, points []model.MonitoringPoint, measurements []model.NoiseMeasurement, profiles []model.SourceProfile, createdBy uint) error {
	measurementInputs := make([]algorithm.MeasurementInput, 0, len(measurements))
	for index, measurement := range measurements {
		var bands, background algorithm.Spectrum
		if err := json.Unmarshal([]byte(measurement.OctaveBandsJSON), &bands); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(points[index].BackgroundProfileJSON), &background); err != nil {
			return err
		}
		measurementInputs = append(measurementInputs, algorithm.MeasurementInput{
			ID: measurement.ID, Checksum: measurement.SourceChecksum, Bands: bands,
			Point: algorithm.PointInput{ID: points[index].ID, PointCode: points[index].PointCode, XM: points[index].XM, YM: points[index].YM, HeightM: points[index].HeightM, Background: background},
		})
	}
	sourceInputs := make([]algorithm.SourceInput, 0, len(profiles))
	for _, profile := range profiles {
		var power, directivity algorithm.Spectrum
		if err := json.Unmarshal([]byte(profile.OctavePowerJSON), &power); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(profile.DirectivityJSON), &directivity); err != nil {
			return err
		}
		sourceInputs = append(sourceInputs, algorithm.SourceInput{
			ID: profile.ID, SourceCode: profile.SourceCode, Name: profile.Name,
			XM: profile.XM, YM: profile.YM, HeightM: profile.HeightM,
			ReferenceDistanceM: profile.ReferenceDistanceM, Power: power, Directivity: directivity,
			OperatingFactor: profile.OperatingFactor, Version: profile.Version,
		})
	}
	snapshot := struct {
		AlgorithmVersion string                       `json:"algorithm_version"`
		Measurements     []algorithm.MeasurementInput `json:"measurements"`
		Sources          []algorithm.SourceInput      `json:"sources"`
	}{constants.AlgorithmVersion, measurementInputs, sourceInputs}
	hash, snapshotJSON, err := algorithm.CanonicalHash(snapshot)
	if err != nil {
		return err
	}
	fit, err := algorithm.FitAttribution(measurementInputs, sourceInputs)
	if err != nil {
		return err
	}
	finished := time.Now().UTC().Add(-20 * time.Hour)
	run := model.AttributionRun{
		RunCode: "AR-SEED-0538", MeasurementIDsJSON: mustSeedJSON([]uint{measurements[0].ID, measurements[1].ID, measurements[2].ID}),
		SourceProfileIDsJSON: mustSeedJSON([]uint{profiles[0].ID, profiles[1].ID, profiles[2].ID}),
		AlgorithmVersion:     constants.AlgorithmVersion, InputHash: hash, InputSnapshotJSON: string(snapshotJSON),
		NormalizedBandsJSON: mustSeedJSON(fit.NormalizedBands), ContributionsJSON: mustSeedJSON(fit.Contributions), EvidenceJSON: mustSeedJSON(fit.Evidence),
		ResidualError: fit.ResidualError, AttributionState: string(constants.AttributionCompleted), Explanation: fit.Explanation,
		StartedAt: finished.Add(-time.Duration(fit.Evidence.ElapsedMillis) * time.Millisecond), FinishedAt: &finished,
		CreatedBy: createdBy, Version: 3, CreatedAt: finished, UpdatedAt: finished,
	}
	if err := tx.Create(&run).Error; err != nil {
		return err
	}
	return tx.Create(seedAudit(createdBy, "Noise Data Analyst", "attribution_run.completed", "AttributionRun", run.ID, run, map[string]any{
		"input_hash": hash, "algorithm_version": constants.AlgorithmVersion,
		"matrix_rows": fit.Evidence.MatrixRows, "matrix_columns": fit.Evidence.MatrixColumns,
	})).Error
}

func seedAudit(actorID uint, actorName, action, entityType string, entityID uint, after, metadata any) *model.AuditLog {
	return &model.AuditLog{
		RequestID: fmt.Sprintf("seed-%s-%03d", strings.ToLower(entityType), entityID),
		ActorID:   actorID, ActorName: actorName, Action: action, EntityType: entityType, EntityID: entityID,
		BeforeJSON: "{}", AfterJSON: mustSeedJSON(after), MetadataJSON: mustSeedJSON(metadata), CreatedAt: time.Now().UTC(),
	}
}

func flatDirection(value float64) algorithm.Spectrum {
	result := make(algorithm.Spectrum, len(constants.OctaveBands))
	for _, band := range constants.OctaveBands {
		result[algorithm.BandKey(band)] = value
	}
	return result
}

func mustSeedJSON(value any) string {
	_, payload, err := algorithm.CanonicalHash(value)
	if err != nil {
		panic(err)
	}
	return string(payload)
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
