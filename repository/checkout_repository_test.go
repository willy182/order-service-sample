package repository

import (
	"database/sql"
	"errors"
	"order-service-sample/model"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetProductPrice(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name          string
		productID     int
		mockSetup     func()
		expectedPrice int64
		expectedError string
	}{
		{
			name:      "Product found successfully",
			productID: 1,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"price"}).
					AddRow(2999)
				mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedPrice: 2999,
			expectedError: "",
		},
		{
			name:      "Product not found - returns product_not_found error",
			productID: 999,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedPrice: 0,
			expectedError: "product_not_found",
		},
		{
			name:      "Database connection error",
			productID: 1,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
					WithArgs(1).
					WillReturnError(errors.New("connection failed"))
			},
			expectedPrice: 0,
			expectedError: "connection failed",
		},
		{
			name:      "Invalid price type in database",
			productID: 2,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"price"}).
					AddRow("invalid") // Wrong type
				mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedPrice: 0,
			expectedError: "sql: Scan error", // Partial match for scan error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.mockSetup()

			// Execute the function
			price, err := GetProductPrice(db, tt.productID)

			// Verify the price
			if price != tt.expectedPrice {
				t.Errorf("GetProductPrice() price = %d, expected %d", price, tt.expectedPrice)
			}

			// Verify the error
			if err != nil {
				if tt.expectedError == "" {
					t.Errorf("GetProductPrice() unexpected error = %v", err)
				} else if err.Error() != tt.expectedError &&
					!contains(err.Error(), tt.expectedError) {
					t.Errorf("GetProductPrice() error = %v, expected to contain %v", err.Error(), tt.expectedError)
				}
			} else if tt.expectedError != "" {
				t.Errorf("GetProductPrice() expected error = %v, got nil", tt.expectedError)
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

// Helper function to check if error message contains expected string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > 0 && len(substr) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

func TestGetProductPrice_EdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Test with zero product ID
	t.Run("Zero product ID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
			WithArgs(0).
			WillReturnError(sql.ErrNoRows)

		price, err := GetProductPrice(db, 0)

		if price != 0 {
			t.Errorf("Expected price 0 for non-existent product, got %d", price)
		}
		if err == nil || err.Error() != "product_not_found" {
			t.Errorf("Expected 'product_not_found' error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	// Test with negative product ID
	t.Run("Negative product ID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT price FROM products WHERE id=\$1`).
			WithArgs(-1).
			WillReturnError(sql.ErrNoRows)

		price, err := GetProductPrice(db, -1)

		if price != 0 {
			t.Errorf("Expected price 0 for non-existent product, got %d", price)
		}
		if err == nil || err.Error() != "product_not_found" {
			t.Errorf("Expected 'product_not_found' error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCreateOrder_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders (user_id, total_amount, status)
		VALUES ($1, $2, 'pending') RETURNING id`)).
		WithArgs(10, int64(1000)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))

	id, err := CreateOrder(db, 10, int64(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected id 123, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestInsertOrderItem_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	item := struct {
		ProductID int
		Qty       int
	}{ProductID: 2, Qty: 3}
	// we assume domain.OrderItem has ProductID and Qty fields;
	// but InsertOrderItem signature we used: (db *sql.DB, orderID int, item domain.OrderItem, price int64)
	// in test we call InsertOrderItem with a simplified variant: pass values through same query expectation

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO order_items (order_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)`)).
		WithArgs(10, item.ProductID, item.Qty, int64(5000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Call the repo function
	// Since InsertOrderItem in your repo expects domain.OrderItem, adapt this call to your actual signature.
	// For the test, we call the function that exists in your repo:
	err := InsertOrderItem(db, 10, model.CheckoutItem{
		ProductID: item.ProductID, Qty: item.Qty,
	}, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
