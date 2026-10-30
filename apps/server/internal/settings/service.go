// Package settings keeps the server's site-wide settings: values admins change
// in the admin panel, such as whether sign-ups are open.
//
// Each feature declares its own settings in code (Bool, Number, Text) with a
// type, a default, limits and a description, and passes them to NewService at
// startup. The settings table stores only the values admins have set. Changes
// apply at once, without a restart: the service keeps the current values in
// memory and updates them as it writes them.
package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrUnknownSetting is returned for a key no feature has declared.
var ErrUnknownSetting = errors.New("unknown setting")

// InvalidValueError is returned for a value a setting does not accept.
type InvalidValueError struct {
	Key    string
	Value  string
	Reason string
}

func (e *InvalidValueError) Error() string {
	return fmt.Sprintf("setting %q: value %q %s", e.Key, e.Value, e.Reason)
}

// Entry is a setting with its current value, as listed in the admin panel.
type Entry struct {
	Info
	Value string
	// IsDefault is true while no admin has set the setting.
	IsDefault bool
	// UpdatedAt is when an admin last set the setting; zero while IsDefault.
	UpdatedAt time.Time
}

// settingRow is a row of the settings table.
type settingRow struct {
	Key       string `gorm:"primaryKey"`
	Value     string
	UpdatedAt time.Time
}

func (settingRow) TableName() string { return "settings" }

// current is a setting's value in memory.
type current struct {
	raw       string
	value     any
	isDefault bool
	updatedAt time.Time
}

// Service reads and changes the settings. It is safe for concurrent use.
type Service struct {
	db          *gorm.DB
	logger      *slog.Logger
	definitions map[string]Definition
	// order lists the keys as declared, which is the order List returns.
	order []string

	// writeMu serialises Set and Reset, so the database and the values in
	// memory always agree about the latest change.
	writeMu sync.Mutex
	// mu guards values.
	mu     sync.RWMutex
	values map[string]current
}

// NewService checks the definitions and loads the values admins have set. A
// stored value no longer declared, or no longer valid for its definition, is
// logged and ignored, so the setting falls back to its default; it is not
// deleted, in case a later version declares it again.
func NewService(ctx context.Context, db *gorm.DB, logger *slog.Logger, definitions ...Definition) (*Service, error) {
	s := &Service{
		db:          db,
		logger:      logger,
		definitions: make(map[string]Definition, len(definitions)),
		order:       make([]string, 0, len(definitions)),
		values:      make(map[string]current, len(definitions)),
	}

	var problems []error
	for _, definition := range definitions {
		if err := check(definition); err != nil {
			problems = append(problems, err)
			continue
		}

		key := definition.Describe().Key
		if _, taken := s.definitions[key]; taken {
			problems = append(problems, fmt.Errorf("setting %q is declared twice", key))
			continue
		}

		s.definitions[key] = definition
		s.order = append(s.order, key)
		s.values[key] = defaultValue(definition)
	}
	if err := errors.Join(problems...); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	var stored []settingRow
	if err := db.WithContext(ctx).Find(&stored).Error; err != nil {
		return nil, fmt.Errorf("loading settings: %w", err)
	}

	for _, row := range stored {
		definition, known := s.definitions[row.Key]
		if !known {
			logger.Warn("ignoring a stored setting no feature declares", "key", row.Key)
			continue
		}

		raw, value, err := definition.canonical(row.Value)
		if err != nil {
			logger.Warn("ignoring a stored setting value that is no longer valid; using the default", "key", row.Key, "value", row.Value, "reason", err.Error())
			continue
		}

		s.values[row.Key] = current{raw: raw, value: value, updatedAt: row.UpdatedAt}
	}

	return s, nil
}

func defaultValue(definition Definition) current {
	// check has already proven the default valid.
	raw, value, _ := definition.canonical(definition.Describe().Default)

	return current{raw: raw, value: value, isDefault: true}
}

// Bool returns the current value of a declared Bool setting. It panics if the
// setting was not passed to NewService, which is a programming error.
func (s *Service) Bool(definition Bool) bool {
	return s.get(definition).(bool)
}

// Number returns the current value of a declared Number setting. It panics if
// the setting was not passed to NewService, which is a programming error.
func (s *Service) Number(definition Number) int64 {
	return s.get(definition).(int64)
}

// Text returns the current value of a declared Text setting. It panics if the
// setting was not passed to NewService, which is a programming error.
func (s *Service) Text(definition Text) string {
	return s.get(definition).(string)
}

func (s *Service) get(definition Definition) any {
	key := definition.Describe().Key
	if declared, ok := s.definitions[key]; !ok || declared != definition {
		panic(fmt.Sprintf("settings: %q was not declared with this definition", key))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.values[key].value
}

// List returns every setting with its current value, in the order declared.
func (s *Service) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]Entry, 0, len(s.order))
	for _, key := range s.order {
		entries = append(entries, s.entry(key))
	}

	return entries
}

// entry builds a setting's Entry. The caller holds mu.
func (s *Service) entry(key string) Entry {
	value := s.values[key]

	return Entry{Info: s.definitions[key].Describe(), Value: value.raw, IsDefault: value.isDefault, UpdatedAt: value.updatedAt}
}

// Recorder is called in the transaction changing a setting, with its value
// before and after, to record the change with it; an error undoes the change.
type Recorder func(tx *gorm.DB, before, after string) error

// Set checks a value against its setting, stores it and applies it at once.
// It returns ErrUnknownSetting for an undeclared key and an
// *InvalidValueError for a value the setting does not accept.
func (s *Service) Set(ctx context.Context, key, raw string) (Entry, error) {
	return s.SetRecorded(ctx, key, raw, nil)
}

// SetRecorded is Set, with record, if any, called in its transaction.
func (s *Service) SetRecorded(ctx context.Context, key, raw string, record Recorder) (Entry, error) {
	definition, known := s.definitions[key]
	if !known {
		return Entry{}, fmt.Errorf("%w: %q", ErrUnknownSetting, key)
	}

	canonical, value, err := definition.canonical(raw)
	if err != nil {
		return Entry{}, &InvalidValueError{Key: key, Value: raw, Reason: err.Error()}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	row := settingRow{Key: key, Value: canonical, UpdatedAt: s.db.NowFunc()}
	before := s.rawValue(key)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&row).Error
		if err != nil {
			return fmt.Errorf("storing setting %q: %w", key, err)
		}
		if record == nil {
			return nil
		}
		return record(tx, before, canonical)
	})
	if err != nil {
		return Entry{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = current{raw: canonical, value: value, updatedAt: row.UpdatedAt}

	return s.entry(key), nil
}

// rawValue is a setting's current value, as stored.
func (s *Service) rawValue(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[key].raw
}

// Reset removes an admin's value, so the setting goes back to its default. It
// returns ErrUnknownSetting for an undeclared key.
func (s *Service) Reset(ctx context.Context, key string) (Entry, error) {
	return s.ResetRecorded(ctx, key, nil)
}

// ResetRecorded is Reset, with record, if any, called in its transaction.
func (s *Service) ResetRecorded(ctx context.Context, key string, record Recorder) (Entry, error) {
	definition, known := s.definitions[key]
	if !known {
		return Entry{}, fmt.Errorf("%w: %q", ErrUnknownSetting, key)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	before := s.rawValue(key)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&settingRow{Key: key}).Error; err != nil {
			return fmt.Errorf("resetting setting %q: %w", key, err)
		}
		if record == nil {
			return nil
		}
		return record(tx, before, definition.Describe().Default)
	})
	if err != nil {
		return Entry{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = defaultValue(definition)

	return s.entry(key), nil
}
