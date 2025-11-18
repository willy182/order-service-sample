package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"order-service-sample/model"
)

type User struct {
	ID           int
	Email        string
	Phone        string
	PasswordHash string
}

func GetUserByEmail(db *sql.DB, email string) (User, error) {
	var u User
	row := db.QueryRow(`SELECT id, email, phone, password_hash FROM users WHERE email = $1`, email)
	err := row.Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash)
	return u, err
}

func GetUserByPhone(db *sql.DB, phone string) (User, error) {
	var u User
	row := db.QueryRow(`SELECT id, email, phone, password_hash FROM users WHERE phone = $1`, phone)
	err := row.Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash)
	return u, err
}

// ReleaseReservationByOrderID releases reserved stock when reservation expires.
func ReleaseReservationByOrderID(db *sql.DB, orderID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Step 1: Ambil data reservation berdasarkan order_id
	rows, err := tx.Query(`
		SELECT product_id, warehouse_id, quantity
		FROM reservations
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type reservationItem struct {
		ProductID   int
		WarehouseID int
		Qty         int
	}

	var items []reservationItem
	for rows.Next() {
		var item reservationItem
		if err := rows.Scan(&item.ProductID, &item.WarehouseID, &item.Qty); err != nil {
			return err
		}
		items = append(items, item)
	}

	// Step 2: Update stok (kurangi reserved)
	for _, item := range items {
		_, err := tx.Exec(`
			UPDATE warehouse_stock
			SET reserved = GREATEST(reserved - $1, 0)
			WHERE product_id = $2 AND warehouse_id = $3
		`, item.Qty, item.ProductID, item.WarehouseID)
		if err != nil {
			return err
		}
		log.Printf("[worker] released %d reserved stock for product_id=%d in warehouse_id=%d", item.Qty, item.ProductID, item.WarehouseID)
	}

	// Step 3: Hapus data reservation terkait order ini
	_, err = tx.Exec(`DELETE FROM reservations WHERE order_id = $1`, orderID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// ReserveStockForOrder reserves stock for an order by finding an active warehouse.
func ReserveStockForOrder(ctx context.Context, db *sql.DB, orderID int, items []model.CheckoutItem) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {

		// 1. Cari warehouse aktif yang punya stok cukup
		var warehouseID int
		var stock, reserved int

		err := tx.QueryRow(`
			SELECT ws.warehouse_id, ws.quantity, ws.reserved
			FROM warehouse_stock ws
			JOIN warehouses w ON w.id = ws.warehouse_id
			WHERE w.active = TRUE
			AND ws.product_id = $1
			AND (ws.quantity - ws.reserved) >= $2
			ORDER BY ws.warehouse_id
			LIMIT 1
		`, item.ProductID, item.Qty).Scan(&warehouseID, &stock, &reserved)

		if err == sql.ErrNoRows {
			return fmt.Errorf("no active warehouse has enough stock for product %d", item.ProductID)
		}
		if err != nil {
			return err
		}

		// 2. Update reserved di warehouse_stock
		_, err = tx.Exec(`
			UPDATE warehouse_stock
			SET reserved = reserved + $1
			WHERE warehouse_id = $2 AND product_id = $3
		`, item.Qty, warehouseID, item.ProductID)
		if err != nil {
			return err
		}

		// 3. Update stock di products
		_, err = tx.Exec(`
			UPDATE products
			SET stock = stock - $1
			WHERE id = $2
		`, item.Qty, item.ProductID)
		if err != nil {
			return err
		}

		// 4. Buat record di reservations
		_, err = tx.Exec(`
			INSERT INTO reservations (order_id, product_id, warehouse_id, quantity, expires_at)
			VALUES ($1, $2, $3, $4, NOW() + INTERVAL '5 minutes')
		`, orderID, item.ProductID, warehouseID, item.Qty)
		if err != nil {
			return err
		}

		log.Printf("[checkout] reserved %d units of product_id=%d in warehouse_id=%d",
			item.Qty, item.ProductID, warehouseID)
	}

	return tx.Commit()
}
