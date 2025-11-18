package repository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetAllProducts_SuccessSingleRow(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "name", "stock", "price", "description",
	}).AddRow(1, "Product A", 10, 5000, "Desc A")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDER BY id
	`)).WillReturnRows(rows)

	products, err := GetAllProducts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].Name != "Product A" {
		t.Fatalf("unexpected product name: %s", products[0].Name)
	}
}

func TestGetAllProducts_SuccessMultipleRows(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "name", "stock", "price", "description",
	}).
		AddRow(1, "Product A", 10, 5000, "Desc A").
		AddRow(2, "Product B", 3, 9999, "Desc B")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDER BY id
	`)).WillReturnRows(rows)

	products, err := GetAllProducts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[1].Name != "Product B" {
		t.Fatalf("unexpected product name: %s", products[1].Name)
	}
}

func TestGetAllProducts_EmptyResult(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "name", "stock", "price", "description",
	}) // no AddRow → empty

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDER BY id
	`)).WillReturnRows(rows)

	products, err := GetAllProducts(db)
	if err != nil {
		t.Fatalf("unexpected empty result error: %v", err)
	}

	if len(products) != 0 {
		t.Fatalf("expected 0 products, got %d", len(products))
	}
}

func TestGetAllProducts_QueryError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDERORDER id
	`)).WillReturnError(errors.New("db failure"))

	_, err := GetAllProducts(db)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestGetAllProducts_ScanError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// returning invalid string for stock (expected int)
	rows := sqlmock.NewRows([]string{
		"id", "name", "stock", "price", "description",
	}).AddRow(1, "Product A", "NOT_INT", 5000, "Desc A")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDER BY id
	`)).WillReturnRows(rows)

	_, err := GetAllProducts(db)
	if err == nil {
		t.Fatalf("expected scan error but got nil")
	}
}

func TestGetAllProducts_RowsNextError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "name", "stock", "price", "description",
	}).AddRow(1, "Prod X", 10, 1000, "Desc X").
		RowError(0, errors.New("row next error"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, stock, price, description 
		FROM products 
		ORDER BY id
	`)).WillReturnRows(rows)

	_, err := GetAllProducts(db)
	if err == nil {
		t.Fatalf("expected row error but got nil")
	}
}
