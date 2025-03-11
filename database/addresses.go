package database

import (
	"errors"
	"gorm.io/gorm"
	"strings"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
)

type Addresses struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	Address     string    `json:"address"`
	AddressType uint8     `json:"address_type"` //0:用户地址；1:热钱包地址(归集地址)；2:冷钱包地址
	PublicKey   string    `json:"public_key"`
	Timestamp   uint64
}

type AddressesView interface {
	AddressExist(requestId string, address *common.Address) (bool, uint8)
	QueryAddressesByToAddress(string, *common.Address) (*Addresses, error)
	QueryHotWalletInfo(string) (*Addresses, error)
	QueryColdWalletInfo(string) (*Addresses, error)
	GetAllAddresses(string) ([]*Addresses, error)
}

type AddressesDB interface {
	AddressesView

	StoreAddresses(string, []Addresses) error
}

type addressesDB struct {
	gorm *gorm.DB
}

func (db *addressesDB) AddressExist(requestId string, address *common.Address) (bool, uint8) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address", strings.ToLower(address.String())).First(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0
		}
		return false, 0
	}
	return true, addressEntry.AddressType
}

func (db *addressesDB) QueryAddressesByToAddress(requestId string, address *common.Address) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address", strings.ToLower(address.String())).Take(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &addressEntry, nil
}

func NewAddressesDB(db *gorm.DB) AddressesDB {
	return &addressesDB{gorm: db}
}

// StoreAddresses store address
func (db *addressesDB) StoreAddresses(requestId string, addressList []Addresses) error {
	result := db.gorm.Table("addresses_"+requestId).CreateInBatches(&addressList, len(addressList))
	return result.Error
}

func (db *addressesDB) QueryHotWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address_type", 1).Take(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (db *addressesDB) QueryColdWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address_type", 2).Take(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (db *addressesDB) GetAllAddresses(requestId string) ([]*Addresses, error) {
	var addresses []*Addresses
	err := db.gorm.Table("addresses_" + requestId).Find(&addresses).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return addresses, nil
}
