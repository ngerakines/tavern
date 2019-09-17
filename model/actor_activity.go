package model

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"time"
)

type ActorActivity struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`

	Actor   Actor     `gorm:"foreignkey:ActorID"`
	ActorID uuid.UUID `gorm:"not null;type:uuid;unique_index:actor_activity"`

	Activity   Activity  `gorm:"foreignkey:ActivityID"`
	ActivityID uuid.UUID `gorm:"not null;type:uuid;unique_index:actor_activity"`

	Public bool `gorm:"not null;default false;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (cs *ActorActivity) BeforeCreate(scope *gorm.Scope) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	return scope.SetColumn("ID", id)
}

func PublicActorActivityCount(db *gorm.DB, actorID uuid.UUID) (int, error) {
	var count int
	err := db.
		Model(&ActorActivity{}).
		Where("actor_id = ? AND public = true", actorID).
		Count(&count).
		Error
	if err != nil {
		return -1, err
	}
	return count, nil
}

func PublicActorActivity(db *gorm.DB, actorID uuid.UUID, page, limit int) ([]ActorActivity, error) {
	var actorActivity []ActorActivity
	err := db.
		Where("actor_id = ? AND public = true", actorID).
		Order("created_at asc").
		Offset((page - 1) * limit).
		Limit(limit).
		Preload("Activity").
		Find(&actorActivity).
		Error
	if err != nil {
		return nil, err
	}
	return actorActivity, nil
}
