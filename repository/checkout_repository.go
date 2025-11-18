package repository

import (
	"database/sql"
	"fmt"
	"order-service-sample/model"
	"strconv"
	"strings"
)

func GetProductPrice(db *sql.DB, productID int) (int64, error) {
	var priceStr string
	err := db.QueryRow(`
		SELECT price 
		FROM products 
		WHERE id=$1
	`, productID).Scan(&priceStr)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("product_not_found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query product price: %w", err)
	}

	// Convert string price to cents (int64)
	// Example: "700000.00" -> 70000000 cents
	priceInCents, err := convertPriceToCents(priceStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert price to cents: %w", err)
	}

	return priceInCents, nil
}

// convertPriceToCents converts a price string to cents (int64)
func convertPriceToCents(priceStr string) (int64, error) {
	// Remove any currency symbols and trim spaces
	priceStr = strings.TrimSpace(priceStr)

	// Parse the decimal string
	parts := strings.Split(priceStr, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid price format: %s", priceStr)
	}

	// Parse dollars part
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid dollars part: %w", err)
	}

	// Parse cents part (if exists)
	var cents int64
	if len(parts) == 2 {
		// Pad or truncate cents to 2 digits
		centsStr := parts[1]
		if len(centsStr) > 2 {
			centsStr = centsStr[:2] // Truncate to 2 digits
		} else if len(centsStr) < 2 {
			centsStr = centsStr + strings.Repeat("0", 2-len(centsStr)) // Pad with zeros
		}

		cents, err = strconv.ParseInt(centsStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid cents part: %w", err)
		}
	}

	// Calculate total cents
	totalCents := dollars*100 + cents
	return totalCents, nil
}

func CreateOrder(db *sql.DB, userID int, totalAmount int64) (int, error) {
	var orderID int

	err := db.QueryRow(`
		INSERT INTO orders (user_id, total_amount, status)
		VALUES ($1, $2, 'pending') RETURNING id
	`, userID, totalAmount).Scan(&orderID)

	return orderID, err
}

func InsertOrderItem(db *sql.DB, orderID int, item model.CheckoutItem, price int64) error {
	_, err := db.Exec(`
		INSERT INTO order_items (order_id, product_id, quantity, price)
		VALUES ($1, $2, $3, $4)
	`, orderID, item.ProductID, item.Qty, price)

	return err
}
