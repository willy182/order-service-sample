package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetOrderReservationItems_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"product_id", "warehouse_id", "quantity"}).
		AddRow(1, 1, 2).
		AddRow(2, 1, 1)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT product_id, warehouse_id, quantity
		FROM reservations
		WHERE order_id = $1
	`)).WithArgs(7).WillReturnRows(rows)

	items, err := GetOrderReservationItems(db, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ProductID != 1 || items[0].WarehouseID != 1 || items[0].Qty != 2 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestApplyStockPayment_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// create tx
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
			UPDATE warehouse_stock
			SET quantity = quantity - $1,
			    reserved = reserved - $1,
			    updated_at = NOW()
			WHERE warehouse_id = $2 AND product_id = $3
		`)).
		WithArgs(2, 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	items := []ReservationItem{
		{ProductID: 1, WarehouseID: 1, Qty: 2},
	}
	if err := ApplyStockPayment(tx, items); err != nil {
		t.Fatalf("apply stock payment err: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestClearReservation_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	// Begin transaction
	mock.ExpectBegin()

	// Expect DELETE statement
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM reservations WHERE order_id = $1`)).
		WithArgs(55).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect commit
	mock.ExpectCommit()

	tx, _ := db.Begin()
	err = ClearReservation(tx, 55)
	if err != nil {
		t.Fatalf("ClearReservation returned error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestClearReservation_Error(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	// Query returns error
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM reservations WHERE order_id = $1`)).
		WithArgs(999).
		WillReturnError(sqlmock.ErrCancelled)

	tx, _ := db.Begin()
	err := ClearReservation(tx, 999)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}

	// rollback because Exec failed
	tx.Rollback()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateOrderPaid_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
	`)).
		WithArgs(77).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	tx, _ := db.Begin()
	err := UpdateOrderPaid(tx, 77)
	if err != nil {
		t.Fatalf("UpdateOrderPaid returned error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateOrderPaid_Error(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
	`)).
		WithArgs(888).
		WillReturnError(sqlmock.ErrCancelled)

	tx, _ := db.Begin()
	err := UpdateOrderPaid(tx, 888)
	if err == nil {
		t.Fatalf("expected error from UpdateOrderPaid")
	}

	tx.Rollback()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestValidateOrderOwnership(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name           string
		orderID        int
		userID         int
		mockSetup      func()
		expectedStatus string
		expectedError  string
	}{
		{
			name:    "Order found and user is owner - success",
			orderID: 123,
			userID:  456,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"user_id", "status"}).
					AddRow(456, "pending")
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedStatus: "pending",
			expectedError:  "",
		},
		{
			name:    "Order not found - returns order_not_found error",
			orderID: 999,
			userID:  456,
			mockSetup: func() {
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedStatus: "",
			expectedError:  "order_not_found",
		},
		{
			name:    "Database connection error",
			orderID: 123,
			userID:  456,
			mockSetup: func() {
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(123).
					WillReturnError(errors.New("connection failed"))
			},
			expectedStatus: "",
			expectedError:  "connection failed",
		},
		{
			name:    "User is not the owner - returns forbidden error",
			orderID: 123,
			userID:  999, // Different user
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"user_id", "status"}).
					AddRow(456, "pending") // Order belongs to user 456
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedStatus: "",
			expectedError:  "forbidden",
		},
		{
			name:    "Scan error - invalid data types",
			orderID: 123,
			userID:  456,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"user_id", "status"}).
					AddRow("invalid", "pending") // user_id as string instead of int
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedStatus: "",
			expectedError:  "sql: Scan error", // Partial match for scan error
		},
		{
			name:    "Order with different status",
			orderID: 123,
			userID:  456,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"user_id", "status"}).
					AddRow(456, "paid")
				mock.ExpectQuery(`
					SELECT user_id, status
					FROM orders
					WHERE id = \$1
				`).
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedStatus: "paid",
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.mockSetup()

			// Execute the function
			status, err := ValidateOrderOwnership(db, tt.orderID, tt.userID)

			// Verify the status
			if status != tt.expectedStatus {
				t.Errorf("ValidateOrderOwnership() status = %s, expected %s", status, tt.expectedStatus)
			}

			// Verify the error
			if err != nil {
				if tt.expectedError == "" {
					t.Errorf("ValidateOrderOwnership() unexpected error = %v", err)
				} else if err.Error() != tt.expectedError &&
					!contains(err.Error(), tt.expectedError) {
					t.Errorf("ValidateOrderOwnership() error = %v, expected to contain %v", err.Error(), tt.expectedError)
				}
			} else if tt.expectedError != "" {
				t.Errorf("ValidateOrderOwnership() expected error = %v, got nil", tt.expectedError)
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestValidateOrderOwnership_EdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Test with zero IDs
	t.Run("Zero order ID and user ID", func(t *testing.T) {
		mock.ExpectQuery(`
			SELECT user_id, status
			FROM orders
			WHERE id = \$1
		`).
			WithArgs(0).
			WillReturnError(sql.ErrNoRows)

		status, err := ValidateOrderOwnership(db, 0, 0)

		if status != "" {
			t.Errorf("Expected empty status for non-existent order, got %s", status)
		}
		if err == nil || err.Error() != "order_not_found" {
			t.Errorf("Expected 'order_not_found' error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	// Test with negative IDs
	t.Run("Negative order ID", func(t *testing.T) {
		mock.ExpectQuery(`
			SELECT user_id, status
			FROM orders
			WHERE id = \$1
		`).
			WithArgs(-1).
			WillReturnError(sql.ErrNoRows)

		status, err := ValidateOrderOwnership(db, -1, 456)

		if status != "" {
			t.Errorf("Expected empty status for non-existent order, got %s", status)
		}
		if err == nil || err.Error() != "order_not_found" {
			t.Errorf("Expected 'order_not_found' error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	// Test same user ID as owner (edge case for ownership check)
	t.Run("Same user ID as owner", func(t *testing.T) {
		userID := 456
		rows := sqlmock.NewRows([]string{"user_id", "status"}).
			AddRow(userID, "pending")
		mock.ExpectQuery(`
			SELECT user_id, status
			FROM orders
			WHERE id = \$1
		`).
			WithArgs(123).
			WillReturnRows(rows)

		status, err := ValidateOrderOwnership(db, 123, userID)

		if err != nil {
			t.Errorf("Expected no error for same user ID, got %v", err)
		}
		if status != "pending" {
			t.Errorf("Expected status 'pending', got %s", status)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
