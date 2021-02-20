package model

import "github.com/jinzhu/gorm"

type TransactionScopedWork func(db *gorm.DB) error

func RunTransactionWithOptions(db *gorm.DB, txBody TransactionScopedWork) error {
	tx := db.Begin()

	err := txBody(tx)
	if err != nil {
		if txErr := tx.Rollback().Error; txErr != nil {
			return txErr
		}
		return err
	}
	return tx.Commit().Error
}
