package model

type LoginReq struct {
	EmailOrPhone string `json:"email_or_phone"`
	Password     string `json:"password"`
}

type ProductResp struct {
	ID          int    `json:"id"`
	Price       string `json:"price"`
	Name        string `json:"name"`
	Stock       int    `json:"stock"`
	Description string `json:"description"`
}

type CheckoutItem struct {
	ProductID int `json:"product_id"`
	Qty       int `json:"qty"`
}

type CheckoutRequest struct {
	Items  []CheckoutItem `json:"items"`
	UserID string         `json:"-"`
}

type TransferReq struct {
	ProductID     int `json:"product_id"`
	FromWarehouse int `json:"from_warehouse_id"`
	ToWarehouse   int `json:"to_warehouse_id"`
	Quantity      int `json:"quantity"`
}

type CheckoutResponse struct {
	OrderID int `json:"order_id"`
}

type PayRequest struct {
	OrderID int `json:"order_id"`
}

type PayResponse struct {
	OrderID int    `json:"order_id"`
	Status  string `json:"status"`
}

type UpdateWarehouseStatusReq struct {
	Status string `json:"status"` // "active" | "deactive"
}
