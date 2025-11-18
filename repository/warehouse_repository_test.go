package repository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// --------------------------------------------------
// TEST WarehouseExists
// --------------------------------------------------

func TestWarehouseExists_True(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)
	`)

	mock.
		ExpectQuery(query).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := WarehouseExists(db, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true, got false")
	}
}

func TestWarehouseExists_False(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)
	`)

	mock.
		ExpectQuery(query).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := WarehouseExists(db, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatalf("expected exists=false, got true")
	}
}

func TestWarehouseExists_QueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)
	`)

	mock.
		ExpectQuery(query).
		WithArgs(10).
		WillReturnError(errors.New("db_failed"))

	_, err := WarehouseExists(db, 10)
	if err == nil || err.Error() != "db_failed" {
		t.Fatalf("expected db_failed, got: %v", err)
	}
}

// --------------------------------------------------
// TEST UpdateWarehouseStatus
// --------------------------------------------------

func TestUpdateWarehouseStatus_InvalidStatus(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	err := UpdateWarehouseStatus(db, 1, "WRONG")
	if err == nil || err.Error() != "invalid_status" {
		t.Fatalf("expected invalid_status, got: %v", err)
	}
}

func TestUpdateWarehouseStatus_Active_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`)

	mock.
		ExpectExec(query).
		WithArgs(true, 5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := UpdateWarehouseStatus(db, 5, "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateWarehouseStatus_Deactive_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`)

	mock.
		ExpectExec(query).
		WithArgs(false, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := UpdateWarehouseStatus(db, 2, "deactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateWarehouseStatus_UpdateError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`)

	mock.
		ExpectExec(query).
		WithArgs(true, 99).
		WillReturnError(errors.New("update_failed"))

	err := UpdateWarehouseStatus(db, 99, "active")
	if err == nil || err.Error() != "update_failed" {
		t.Fatalf("expected update_failed, got %v", err)
	}
}

func TestUpdateWarehouseStatus_WarehouseNotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`)

	// RowsAffected = 0 → warehouse tidak ditemukan
	mock.
		ExpectExec(query).
		WithArgs(false, 50).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := UpdateWarehouseStatus(db, 50, "deactive")
	if err == nil || err.Error() != "warehouse_not_found" {
		t.Fatalf("expected warehouse_not_found, got %v", err)
	}
}

func TestUpdateWarehouseStatus_RowsAffectedError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	query := regexp.QuoteMeta(`
		UPDATE warehouses
		SET active = $1
		WHERE id = $2
	`)

	mock.
		ExpectExec(query).
		WithArgs(true, 7).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows_failed")))

	err := UpdateWarehouseStatus(db, 7, "active")
	if err == nil || err.Error() != "rows_failed" {
		t.Fatalf("expected rows_failed, got %v", err)
	}
}
