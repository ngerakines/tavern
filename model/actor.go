package model

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"time"
)

type Actor struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name      string    `gorm:"not null;unique_index:actors_name"`
	Domain    string    `gorm:"not null"`
	Key       string    `gorm:"not null;type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ActorID string

func (c *Actor) BeforeCreate(scope *gorm.Scope) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	return scope.SetColumn("ID", id)
}

func (ID ActorID) Followers() string {
	return fmt.Sprintf("%s/followers", ID)
}

func (ID ActorID) FollowersPage(page int) string {
	return fmt.Sprintf("%s/followers?page=%d", ID, page)
}

func (ID ActorID) Following() string {
	return fmt.Sprintf("%s/following", ID)
}

func (ID ActorID) Outbox() string {
	return fmt.Sprintf("%s/outbox", ID)
}

func (ID ActorID) Inbox() string {
	return fmt.Sprintf("%s/inbox", ID)
}

func NewActorID(name, domain string) ActorID {
	return ActorID(fmt.Sprintf("https://%s/users/%s", domain, name))
}

// TODO: Cache this
func ActorLookup(db *gorm.DB, name, domain string) (bool, error) {
	var userCount int64
	err := db.
		Model(&Actor{}).
		Where("name = ? AND domain = ?", name, domain).
		Count(&userCount).
		Error
	if err != nil {
		return false, err
	}
	return userCount == 1, nil
}

func GenerateKey() (string, error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", err
	}

	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	buf := new(bytes.Buffer)
	err = pem.Encode(buf, privateKey)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func CreateActor(db *gorm.DB, name, domain string) (*Actor, error) {
	actor := &Actor{}
	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	err = db.
		Where(Actor{Name: name, Domain: domain}).
		Attrs(Actor{Key: key}).
		FirstOrCreate(&actor).
		Error
	if err != nil {
		return nil, err
	}
	return actor, nil
}
