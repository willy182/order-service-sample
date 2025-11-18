package repository

import (
	"database/sql"
	"errors"
)

type ReservationItem struct {
	ProductID   int
	WarehouseID int
	Qty         int
}

func ValidateOrderOwnership(db *sql.DB, orderID int, userID int) (string, error) {
	var status string
	var ownerID int

	err := db.QueryRow(`
		SELECT user_id, status
		FROM orders
		WHERE id = $1
	`, orderID).Scan(&ownerID, &status)

	if err == sql.ErrNoRows {
		return "", errors.New("order_not_found")
	}
	if err != nil {
		return "", err
	}

	if ownerID != userID {
		return "", errors.New("forbidden")
	}

	return status, nil
}

func GetOrderReservationItems(db *sql.DB, orderID int) ([]ReservationItem, error) {
	rows, err := db.Query(`
		SELECT product_id, warehouse_id, quantity
		FROM reservations
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ReservationItem{}

	for rows.Next() {
		var it ReservationItem
		err := rows.Scan(&it.ProductID, &it.WarehouseID, &it.Qty)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}

	return items, nil
}

func ApplyStockPayment(tx *sql.Tx, items []ReservationItem) error {
	for _, it := range items {
		_, err := tx.Exec(`
			UPDATE warehouse_stock
			SET quantity = quantity - $1,
			    reserved = reserved - $1,
			    updated_at = NOW()
			WHERE warehouse_id = $2 AND product_id = $3
		`, it.Qty, it.WarehouseID, it.ProductID)

		if err != nil {
			return err
		}
	}
	return nil
}

func ClearReservation(tx *sql.Tx, orderID int) error {
	_, err := tx.Exec(`DELETE FROM reservations WHERE order_id = $1`, orderID)
	return err
}

func UpdateOrderPaid(tx *sql.Tx, orderID int) error {
	_, err := tx.Exec(`
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
	`, orderID)
	return err
}
