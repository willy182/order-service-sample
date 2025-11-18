package repository

import (
	"database/sql"
	"errors"
	"fmt"
)

// CheckWarehouseActive returns true if warehouse exists & active
func CheckWarehouseActive(db *sql.DB, warehouseID int) (bool, error) {
	var active bool
	err := db.QueryRow(`SELECT active FROM warehouses WHERE id = $1`, warehouseID).Scan(&active)
	if err == sql.ErrNoRows {
		return false, errors.New("warehouse_not_found")
	}
	if err != nil {
		return false, err
	}
	return active, nil
}

// GetAvailableStock returns available stock = quantity - reserved for a product in a warehouse
func GetAvailableStock(db *sql.DB, warehouseID, productID int) (int, error) {
	var quantity, reserved int
	err := db.QueryRow(`
		SELECT quantity, reserved
		FROM warehouse_stock
		WHERE warehouse_id = $1 AND product_id = $2
	`, warehouseID, productID).Scan(&quantity, &reserved)

	if err == sql.ErrNoRows {
		return 0, errors.New("stock_not_found")
	}
	if err != nil {
		return 0, err
	}
	avail := quantity - reserved
	if avail < 0 {
		avail = 0
	}
	return avail, nil
}

// ensureStockRowExists ensures a row exists in warehouse_stock
func ensureStockRowExists(tx *sql.Tx, warehouseID, productID int) error {
	q := `INSERT INTO warehouse_stock (warehouse_id, product_id, quantity, reserved)
          VALUES ($1, $2, 0, 0)
          ON CONFLICT (warehouse_id, product_id) DO NOTHING`

	_, err := tx.Exec(q, warehouseID, productID)
	return err
}

// TransferStock transfers qty from source warehouse to destination warehouse
func TransferStock(tx *sql.Tx, fromWarehouseID, toWarehouseID, productID, qty int) error {

	if fromWarehouseID == toWarehouseID {
		return errors.New("from and to warehouse must be different")
	}
	if qty <= 0 {
		return errors.New("quantity must be > 0")
	}

	// Ensure destination stock row exists (so we can update it)
	if err := ensureStockRowExists(tx, toWarehouseID, productID); err != nil {
		return err
	}

	// Lock and read source
	sourceQuery := `SELECT quantity, reserved
                    FROM warehouse_stock
                    WHERE warehouse_id = $1 AND product_id = $2
                    FOR UPDATE`

	var srcQty, srcReserved int
	err := tx.QueryRow(sourceQuery, fromWarehouseID, productID).Scan(&srcQty, &srcReserved)
	if err == sql.ErrNoRows {
		return errors.New("source_stock_not_found")
	}
	if err != nil {
		return err
	}

	available := srcQty - srcReserved
	if available < qty {
		return fmt.Errorf("not enough available stock in source warehouse (available=%d)", available)
	}

	// Lock and read destination
	destQuery := `SELECT quantity, reserved
                  FROM warehouse_stock
                  WHERE warehouse_id = $1 AND product_id = $2
                  FOR UPDATE`

	var dstQty, dstReserved int
	err = tx.QueryRow(destQuery, toWarehouseID, productID).Scan(&dstQty, &dstReserved)
	if err != nil {
		return err
	}

	// Update source
	updateSrc := `UPDATE warehouse_stock
                  SET quantity = quantity - $1
                  WHERE warehouse_id = $2 AND product_id = $3`

	_, err = tx.Exec(updateSrc, qty, fromWarehouseID, productID)
	if err != nil {
		return err
	}

	// Update destination
	updateDst := `UPDATE warehouse_stock
                  SET quantity = quantity + $1
                  WHERE warehouse_id = $2 AND product_id = $3`

	_, err = tx.Exec(updateDst, qty, toWarehouseID, productID)
	if err != nil {
		return err
	}

	return nil
}
