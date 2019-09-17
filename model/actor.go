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
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name       string    `gorm:"not null;unique_index:actors_name"`
	Domain     string    `gorm:"not null"`
	PrivateKey string    `gorm:"not null;type:text;default ''"`
	PublicKey  string    `gorm:"not null;type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ActorID string

func (c *Actor) BeforeCreate(scope *gorm.Scope) error {
	if c.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		return scope.SetColumn("ID", id)
	}
	return nil
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

func (ID ActorID) FollowingPage(page int) string {
	return fmt.Sprintf("%s/following?page=%d", ID, page)
}

func (ID ActorID) Outbox() string {
	return fmt.Sprintf("%s/outbox", ID)
}

func (ID ActorID) OutboxPage(page int) string {
	return fmt.Sprintf("%s/outbox?page=%d", ID, page)
}

func (ID ActorID) Inbox() string {
	return fmt.Sprintf("%s/inbox", ID)
}

func (ID ActorID) MainKey() string {
	return fmt.Sprintf("%s#main-key", ID)
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

func ActorPublicKey(db *gorm.DB, name, domain string) (string, error) {
	var keys []string
	err := db.
		Model(&Actor{}).
		Where("name = ? AND domain = ?", name, domain).
		Pluck("public_key", &keys).
		Error
	if err != nil {
		return "", err
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("no public keys for user")
	}
	return keys[0], nil
}

func ActorUUID(db *gorm.DB, name, domain string) (uuid.UUID, error) {
	var ids []uuid.UUID
	err := db.
		Model(&Actor{}).
		Where("name = ? AND domain = ?", name, domain).
		Pluck("id", &ids).
		Error
	if err != nil {
		return uuid.Nil, err
	}
	if len(ids) == 0 {
		return uuid.Nil, fmt.Errorf("no public keys for user")
	}
	return ids[0], nil
}

func GenerateKey() (string, string, error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", "", err
	}

	privateKey, err := serializePem("PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
	if err != nil {
		return "", "", err
	}

	publicKeyB, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicKey, err := serializePem("PUBLIC KEY", publicKeyB)
	if err != nil {
		return "", "", err
	}

	return privateKey, publicKey, nil
}

func CreateActor(db *gorm.DB, name, domain string) (*Actor, error) {
	actor := &Actor{}
	privateKey, publicKey, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	err = db.
		Where(Actor{Name: name, Domain: domain}).
		Attrs(Actor{PrivateKey: privateKey, PublicKey: publicKey}).
		FirstOrCreate(&actor).
		Error
	if err != nil {
		return nil, err
	}
	return actor, nil
}

func serializePem(pemType string, data []byte) (string, error) {
	var privateKey = &pem.Block{
		Type:  pemType,
		Bytes: data,
	}

	buf := new(bytes.Buffer)
	err := pem.Encode(buf, privateKey)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
