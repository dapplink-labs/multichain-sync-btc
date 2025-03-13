package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Vouts struct {
	GUID      uuid.UUID `gorm:"primaryKey" json:"guid"`
	Address   string    `json:"address"`
	N         uint8     `json:"n"`
	Script    string    `json:"script"`
	Amount    *big.Int  `gorm:"serializer:u256" json:"amount"`
	Timestamp uint64    `json:"timestamp"`
}

type VoutsView interface {
}

type VoutsDB interface {
	VoutsView

	StoreVouts(string, []Vouts) error
}

type voutsDB struct {
	gorm *gorm.DB
}

func NewVoutsDB(db *gorm.DB) VoutsDB {
	return &voutsDB{gorm: db}
}

func (v voutsDB) StoreVouts(businessId string, vouts []Vouts) error {
	//TODO implement me
	panic("implement me")
}
