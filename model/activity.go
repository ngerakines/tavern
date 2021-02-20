package model

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"time"
)

type Activity struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ObjectID  string    `gorm:"type:varchar(10240);not null;unique"`
	Payload   JSON      `gorm:"type:jsonb not null default '{}'::jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Activity) BeforeCreate(scope *gorm.Scope) error {
	if c.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		return scope.SetColumn("ID", id)
	}
	return nil
}
