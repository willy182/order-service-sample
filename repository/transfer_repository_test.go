package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckWarehouseActive_Active(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Row exists and active = true
	rows := sqlmock.NewRows([]string{"active"}).AddRow(true)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT active FROM warehouses WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(rows)

	active, err := CheckWarehouseActive(db, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active {
		t.Fatalf("expected active = true, got false")
	}
}

func TestCheckWarehouseActive_Inactive(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Row exists but active = false
	rows := sqlmock.NewRows([]string{"active"}).AddRow(false)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT active FROM warehouses WHERE id = $1`)).
		WithArgs(2).
		WillReturnRows(rows)

	active, err := CheckWarehouseActive(db, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active {
		t.Fatalf("expected active = false, got true")
	}
}

func TestCheckWarehouseActive_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// No rows → sql.ErrNoRows
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT active FROM warehouses WHERE id = $1`)).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	active, err := CheckWarehouseActive(db, 999)
	if err == nil {
		t.Fatalf("expected warehouse_not_found error but got nil")
	}
	if err.Error() != "warehouse_not_found" {
		t.Fatalf("expected warehouse_not_found, got: %v", err)
	}
	if active {
		t.Fatalf("expected active = false for missing warehouse")
	}
}

func TestCheckWarehouseActive_QueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Unexpected query error
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT active FROM warehouses WHERE id = $1`)).
		WithArgs(55).
		WillReturnError(errors.New("db connection lost"))

	active, err := CheckWarehouseActive(db, 55)
	if err == nil {
		t.Fatalf("expected db error but got nil")
	}
	if active {
		t.Fatalf("expected active = false on error")
	}
}

func TestGetAvailableStock_Normal(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// expected query result: quantity=10, reserved=3 → avail=7
	rows := sqlmock.NewRows([]string{"quantity", "reserved"}).
		AddRow(10, 3)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT quantity, reserved
		FROM warehouse_stock
		WHERE warehouse_id = $1 AND product_id = $2
	`)).
		WithArgs(1, 10).
		WillReturnRows(rows)

	avail, err := GetAvailableStock(db, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avail != 7 {
		t.Fatalf("expected available=7, got %d", avail)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetAvailableStock_NegativeAvailable(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// quantity=5, reserved=10 → avail should become 0
	rows := sqlmock.NewRows([]string{"quantity", "reserved"}).
		AddRow(5, 10)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT quantity, reserved
		FROM warehouse_stock
		WHERE warehouse_id = $1 AND product_id = $2
	`)).
		WithArgs(2, 20).
		WillReturnRows(rows)

	avail, err := GetAvailableStock(db, 2, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avail != 0 {
		t.Fatalf("expected available=0, got %d", avail)
	}
}

func TestGetAvailableStock_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// No rows → SQL ErrNoRows
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT quantity, reserved
		FROM warehouse_stock
		WHERE warehouse_id = $1 AND product_id = $2
	`)).
		WithArgs(3, 30).
		WillReturnError(sql.ErrNoRows)

	avail, err := GetAvailableStock(db, 3, 30)
	if err == nil {
		t.Fatalf("expected stock_not_found error but got nil")
	}
	if err.Error() != "stock_not_found" {
		t.Fatalf("expected stock_not_found, got %v", err)
	}
	if avail != 0 {
		t.Fatalf("expected avail=0 on not found, got %d", avail)
	}
}

func TestGetAvailableStock_QueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Any other SQL error
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT quantity, reserved
		FROM warehouse_stock
		WHERE warehouse_id = $1 AND product_id = $2
	`)).
		WithArgs(4, 40).
		WillReturnError(errors.New("db connection failed"))

	avail, err := GetAvailableStock(db, 4, 40)
	if err == nil {
		t.Fatalf("expected query error but got nil")
	}
	if avail != 0 {
		t.Fatalf("avail should be 0 when error occurs, got %d", avail)
	}
}
