package model

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"time"
)

type Activity struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ActivityType string         `gorm:"not null"`
	Actor        string         `gorm:"not null"`
	Object       ActivityObject `gorm:"type:jsonb not null default '{}'::jsonb"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ActivityObject map[string]interface{}

func (a ActivityObject) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ActivityObject) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

func (c *Activity) BeforeCreate(scope *gorm.Scope) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	return scope.SetColumn("ID", id)
}
