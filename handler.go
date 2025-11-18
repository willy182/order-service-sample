package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"order-service-sample/helper"
	"order-service-sample/model"
	"order-service-sample/repository"

	"github.com/gorilla/mux"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req model.LoginReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user repository.User
	var err error
	if strings.Contains(req.EmailOrPhone, "@") {
		user, err = repository.GetUserByEmail(db, req.EmailOrPhone)
	} else {
		user, err = repository.GetUserByPhone(db, req.EmailOrPhone)
	}

	if err != nil {
		helper.WriteErrorJSON(w, http.StatusUnauthorized, "user not found")
		return
	}

	if !helper.CheckPasswordHash(req.Password, user.PasswordHash) {
		helper.WriteErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := helper.GenerateJWT(user.ID)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	var data = struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

	helper.WriteJSON(w, http.StatusOK, data)
}

func ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	products, err := repository.GetAllProducts(db)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	helper.WriteJSON(w, http.StatusOK, products)
}

func CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// 1. Ambil user_id dari JWT
	userID := helper.GetUserIDFromContext(ctx)

	// 2. Decode JSON
	var req model.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "items cannot be empty")
		return
	}

	// 3. Hitung total harga
	var totalAmount int64
	for _, item := range req.Items {
		price, err := repository.GetProductPrice(db, item.ProductID)
		if err != nil {
			helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		totalAmount += price * int64(item.Qty)
	}

	// 4. Buat order
	orderID, err := repository.CreateOrder(db, userID, totalAmount)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	// 5. Insert order_items
	for _, item := range req.Items {
		price, _ := repository.GetProductPrice(db, item.ProductID)

		err := repository.InsertOrderItem(db, orderID, item, price)
		if err != nil {
			helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to save order items")
			return
		}
	}

	// 6. Reserve stock (versi terbaru yang sudah kamu pakai)
	err = repository.ReserveStockForOrder(ctx, db, orderID, req.Items)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	ttlMinute := helper.ReservationTTLMinutesDefault
	// 7. Set Redis TTL 5 menit
	err = rdb.SetEx(ctx, "reservation:"+fmt.Sprintf("%d", orderID), orderID, time.Duration(ttlMinute)*time.Minute).Err()
	log.Println("error set redis", err)

	// 8. Return JSON
	helper.WriteJSON(w, http.StatusCreated, model.CheckoutResponse{
		OrderID: orderID,
	})
}

func PayHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := helper.GetUserIDFromContext(ctx)

	var req model.PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "order_id required")
		return
	}

	status, err := repository.ValidateOrderOwnership(db, req.OrderID, userID)
	if err != nil {
		if err.Error() == "order_not_found" {
			helper.WriteErrorJSON(w, http.StatusNotFound, "order not found")
			return
		}
		if err.Error() == "forbidden" {
			helper.WriteErrorJSON(w, http.StatusForbidden, "forbidden")
			return
		}
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	if status != "pending" {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "order cannot be paid")
		return
	}

	items, err := repository.GetOrderReservationItems(db, req.OrderID)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to load reservations")
		return
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if err := repository.ApplyStockPayment(tx, items); err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to update stock")
		return
	}

	if err := repository.ClearReservation(tx, req.OrderID); err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to clear reservation")
		return
	}

	if err := repository.UpdateOrderPaid(tx, req.OrderID); err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to update order")
		return
	}

	if err := tx.Commit(); err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "commit failed")
		return
	}

	helper.WriteJSON(w, http.StatusOK, model.PayResponse{
		OrderID: req.OrderID,
		Status:  "paid",
	})
}

func TransferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req model.TransferReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.ProductID == 0 || req.FromWarehouse == 0 || req.ToWarehouse == 0 || req.Quantity <= 0 {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "missing or invalid fields")
		return
	}

	// validate warehouses active
	ok, err := repository.CheckWarehouseActive(db, req.FromWarehouse)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to validate source warehouse")
		return
	}
	if !ok {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "source warehouse not active or not found")
		return
	}

	ok, err = repository.CheckWarehouseActive(db, req.ToWarehouse)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to validate destination warehouse")
		return
	}
	if !ok {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "destination warehouse not active or not found")
		return
	}

	// check available stock quickly (not strictly required because tx will check again)
	avail, err := repository.GetAvailableStock(db, req.FromWarehouse, req.ProductID)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to check stock")
		return
	}
	if avail < req.Quantity {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "not enough available stock in source warehouse")
		return
	}

	// do transfer in transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	if err := repository.TransferStock(tx, req.FromWarehouse, req.ToWarehouse, req.ProductID, req.Quantity); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to commit transfer")
		return
	}

	helper.WriteJSON(w, http.StatusOK, req)
}

func WarehouseUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	warehouseID, err := strconv.Atoi(idStr)
	if err != nil || warehouseID <= 0 {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid warehouse id")
		return
	}

	// Parse body
	var req model.UpdateWarehouseStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Validate enum
	if req.Status != "active" && req.Status != "deactive" {
		helper.WriteErrorJSON(w, http.StatusBadRequest, "status must be 'active' or 'deactive'")
		return
	}

	// Validate warehouse exists
	exists, err := repository.WarehouseExists(db, warehouseID)
	if err != nil {
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !exists {
		helper.WriteErrorJSON(w, http.StatusNotFound, "warehouse not found")
		return
	}

	// Update status
	err = repository.UpdateWarehouseStatus(db, warehouseID, req.Status)
	if err != nil {
		if err.Error() == "warehouse_not_found" {
			helper.WriteErrorJSON(w, http.StatusNotFound, "warehouse not found")
			return
		}
		if err.Error() == "invalid_status" {
			helper.WriteErrorJSON(w, http.StatusBadRequest, "invalid status")
			return
		}
		helper.WriteErrorJSON(w, http.StatusInternalServerError, "failed to update warehouse status")
		return
	}

	// Success response
	helper.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     warehouseID,
		"status": req.Status,
	})
}
