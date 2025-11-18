package repository

import (
	"database/sql"
	"order-service-sample/model"
)

func GetAllProducts(db *sql.DB) ([]model.ProductResp, error) {
	rows, err := db.Query(`
        SELECT id, name, stock, price, description
        FROM products
        ORDER BY id
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.ProductResp

	for rows.Next() {
		var p model.ProductResp
		if err := rows.Scan(&p.ID, &p.Name, &p.Stock, &p.Price, &p.Description); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
