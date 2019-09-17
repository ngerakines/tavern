package model

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"time"
)

type Graph struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Actor     string    `gorm:"not null;unique_index:graph_rel"`
	Follower  string    `gorm:"not null;unique_index:graph_rel"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Graph) TableName() string {
	return "graph"
}

func (c *Graph) BeforeCreate(scope *gorm.Scope) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	return scope.SetColumn("ID", id)
}

func CreateGraphRel(db *gorm.DB, from, to string) (*Graph, error) {
	rel := &Graph{}
	err := db.
		Where(Graph{Actor: to, Follower: from}).
		FirstOrCreate(&rel).
		Error
	if err != nil {
		return nil, err
	}
	return rel, nil
}

func FollowersCount(db *gorm.DB, to string) (int, error) {
	var count int
	err := db.
		Model(&Graph{}).
		Where("actor = ?", to).
		Count(&count).
		Error
	if err != nil {
		return -1, err
	}
	return count, nil
}

// TODO: Cache this
func FollowersPageLookup(db *gorm.DB, to string, page, limit int) ([]string, error) {
	var ids []string
	err := db.
		Model(&Graph{}).
		Order("created_at asc").
		Offset((page-1)*limit).
		Limit(limit).
		Where("actor = ?", to).
		Pluck("follower", &ids).
		Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func FollowingLookup(db *gorm.DB, from string) ([]string, error) {
	var ids []string
	err := db.
		Model(&Graph{}).
		Order("created_at asc").
		Where("follower = ?", from).
		Pluck("actor", &ids).
		Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
