package repository

import (
	"database/sql"
	"errors"
)

func WarehouseExists(db *sql.DB, warehouseID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)
	`, warehouseID).Scan(&exists)

	return exists, err
}

func UpdateWarehouseStatus(db *sql.DB, warehouseID int, status string) error {
	var active bool
	// Gunakan switch agar idiomatic & menghilangkan warning
	switch status {
	case "active":
		active = true
	case "deactive":
		active = false
	default:
		return errors.New("invalid_status")
	}

	res, err := db.Exec(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`, active, warehouseID)

	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("warehouse_not_found")
	}

	return nil
}
