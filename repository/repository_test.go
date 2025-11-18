package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"order-service-sample/model"

	"github.com/DATA-DOG/go-sqlmock"
)

//
// ────────────────────────────────────────────────────────────────
//   GET USER BY EMAIL
// ────────────────────────────────────────────────────────────────
//

func TestGetUserByEmail_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "email", "phone", "password_hash",
	}).AddRow(1, "test@example.com", "08123", "hash")

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, email, phone, password_hash FROM users WHERE email = $1`,
	)).WithArgs("test@example.com").WillReturnRows(rows)

	u, err := GetUserByEmail(db, "test@example.com")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.Email != "test@example.com" {
		t.Fatalf("email mismatch")
	}
}

func TestGetUserByEmail_NoRows(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, email, phone, password_hash FROM users WHERE email = $1`,
	)).
		WithArgs("x@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err := GetUserByEmail(db, "x@example.com")
	if err == nil {
		t.Fatalf("expected error for no rows")
	}
}

//
// ────────────────────────────────────────────────────────────────
//   GET USER BY PHONE
// ────────────────────────────────────────────────────────────────
//

func TestGetUserByPhone_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "email", "phone", "password_hash",
	}).AddRow(1, "p@example.com", "08123", "hash")

	mock.ExpectQuery(`SELECT id, email, phone, password_hash FROM users WHERE phone = \$1`).
		WithArgs("08123").WillReturnRows(rows)

	u, err := GetUserByPhone(db, "08123")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if u.Phone != "08123" {
		t.Fatalf("phone mismatch")
	}
}

func TestGetUserByPhone_NoRows(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(`SELECT id, email, phone, password_hash FROM users WHERE phone = \$1`).
		WithArgs("000").
		WillReturnError(sql.ErrNoRows)

	_, err := GetUserByPhone(db, "000")
	if err == nil {
		t.Fatalf("expected error for no rows")
	}
}

//
// ────────────────────────────────────────────────────────────────
//   RELEASE RESERVATION
// ────────────────────────────────────────────────────────────────
//

func TestReleaseReservationByOrderID_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	// SELECT reservations
	rows := sqlmock.NewRows([]string{"product_id", "warehouse_id", "quantity"}).
		AddRow(101, 1, 5).
		AddRow(102, 1, 3)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT product_id, warehouse_id, quantity
		FROM reservations
		WHERE order_id = $1
	`)).
		WithArgs(99).
		WillReturnRows(rows)

	// UPDATE reserved row 1
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE warehouse_stock
		SET reserved = GREATEST(reserved - $1, 0)
		WHERE product_id = $2 AND warehouse_id = $3
	`)).WithArgs(5, 101, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// UPDATE reserved row 2
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE warehouse_stock
		SET reserved = GREATEST(reserved - $1, 0)
		WHERE product_id = $2 AND warehouse_id = $3
	`)).WithArgs(3, 102, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// DELETE reservations
	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM reservations WHERE order_id = $1`,
	)).
		WithArgs(99).
		WillReturnResult(sqlmock.NewResult(1, 2))

	mock.ExpectCommit()

	err := ReleaseReservationByOrderID(db, 99)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestReleaseReservationByOrderID_EmptyList(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	// SELECT produces empty rows
	rows := sqlmock.NewRows([]string{"product_id", "warehouse_id", "quantity"})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT product_id, warehouse_id, quantity
		FROM reservations
		WHERE order_id = $1
	`)).
		WithArgs(100).
		WillReturnRows(rows)

	// DELETE reservations
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM reservations WHERE order_id = $1`)).
		WithArgs(100).
		WillReturnResult(sqlmock.NewResult(1, 0))

	mock.ExpectCommit()

	err := ReleaseReservationByOrderID(db, 100)
	if err != nil {
		t.Fatalf("unexpected empty-case err: %v", err)
	}
}

func TestReleaseReservationByOrderID_SelectError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT product_id, warehouse_id, quantity FROM reservations WHERE order_id = \$1`).
		WithArgs(1).
		WillReturnError(errors.New("select failed"))

	err := ReleaseReservationByOrderID(db, 1)
	if err == nil {
		t.Fatalf("expected err")
	}
}

func TestReleaseReservationByOrderID_UpdateError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	rows := sqlmock.NewRows([]string{"product_id", "warehouse_id", "quantity"}).
		AddRow(5, 1, 2)

	mock.ExpectQuery(`SELECT product_id, warehouse_id, quantity FROM reservations WHERE order_id = \$1`).
		WithArgs(9).
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE warehouse_stock SET reserved = GREATEST.*`).
		WithArgs(2, 5, 1).
		WillReturnError(errors.New("update fail"))

	err := ReleaseReservationByOrderID(db, 9)
	if err == nil {
		t.Fatalf("expected error on update")
	}
}

//
// ────────────────────────────────────────────────────────────────
//   RESERVE STOCK FOR ORDER
// ────────────────────────────────────────────────────────────────
//

func TestReserveStockForOrder_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	// Step 1: warehouse lookup
	rows := sqlmock.NewRows([]string{
		"warehouse_id", "quantity", "reserved",
	}).AddRow(10, 100, 5)

	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT ws.warehouse_id, ws.quantity, ws.reserved
			FROM warehouse_stock ws
			JOIN warehouses w ON w.id = ws.warehouse_id
			WHERE w.active = TRUE
			  AND ws.product_id = $1
			  AND (ws.quantity - ws.reserved) >= $2
			ORDER BY ws.warehouse_id
			LIMIT 1
		`)).
		WithArgs(101, 2).
		WillReturnRows(rows)

	// Step 2: update reserved
	mock.ExpectExec(regexp.QuoteMeta(`
			UPDATE warehouse_stock
			SET reserved = reserved + $1
			WHERE warehouse_id = $2 AND product_id = $3
		`)).
		WithArgs(2, 10, 101).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Step 3: insert reservation row
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO reservations (order_id, product_id, warehouse_id, quantity, expires_at)
			VALUES ($1, $2, $3, $4, NOW() + INTERVAL '5 minutes')
		`)).
		WithArgs(5000, 101, 10, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := ReserveStockForOrder(ctx, db, 5000, []model.CheckoutItem{
		{ProductID: 101, Qty: 2},
	})
	if err != nil {
		t.Fatalf("unexpected reserve err: %v", err)
	}
}

func TestReserveStockForOrder_NoWarehouseFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT ws.warehouse_id.*`).
		WithArgs(999, 10).
		WillReturnError(sql.ErrNoRows)

	err := ReserveStockForOrder(ctx, db, 2000, []model.CheckoutItem{
		{ProductID: 999, Qty: 10},
	})
	if err == nil {
		t.Fatalf("expected error no active warehouse")
	}
}

func TestReserveStockForOrder_QueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT ws.warehouse_id.*`).
		WithArgs(5, 1).
		WillReturnError(errors.New("query fail"))

	err := ReserveStockForOrder(ctx, db, 9, []model.CheckoutItem{
		{ProductID: 5, Qty: 1},
	})
	if err == nil {
		t.Fatalf("expected query error")
	}
}

func TestReserveStockForOrder_UpdateFail(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	rows := sqlmock.NewRows([]string{"warehouse_id", "quantity", "reserved"}).
		AddRow(3, 10, 1)

	mock.ExpectQuery(`SELECT ws.warehouse_id.*`).
		WithArgs(77, 2).
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE warehouse_stock SET reserved = reserved \+ \$1.*`).
		WithArgs(2, 3, 77).
		WillReturnError(errors.New("update fail"))

	err := ReserveStockForOrder(ctx, db, 7, []model.CheckoutItem{
		{ProductID: 77, Qty: 2},
	})
	if err == nil {
		t.Fatalf("expected update error")
	}
}

func TestReserveStockForOrder_InsertFail(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	rows := sqlmock.NewRows([]string{"warehouse_id", "quantity", "reserved"}).
		AddRow(9, 99, 0)

	mock.ExpectQuery(`SELECT ws.warehouse_id.*`).
		WithArgs(50, 3).
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE warehouse_stock SET reserved = reserved \+ \$1.*`).
		WithArgs(3, 9, 50).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO reservations.*`).
		WithArgs(9999, 50, 9, 3).
		WillReturnError(errors.New("insert fail"))

	err := ReserveStockForOrder(ctx, db, 9999, []model.CheckoutItem{
		{ProductID: 50, Qty: 3},
	})
	if err == nil {
		t.Fatalf("expected insert error")
	}
}
