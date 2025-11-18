# Order Service Sample
A backend service that handles checkout, payments, stock reservations, and warehouse stock transfers.
This application demonstrates a simplified e-commerce backend using real-world architecture principles:

- PostgreSQL for primary storage
- Redis for reservation TTL and expiration worker
- Golang for the backend API and worker
- JWT authentication
- Clean repository pattern
- Unit tests with sqlmock

---

## Features

### Authentication
- Login using **email or phone**
- JWT-based authentication
- All business endpoints require authentication
```curl
curl -X POST http://localhost:8085/login \
  -H "Content-Type: application/json" \
  -d '{"email_or_phone":"admin@example.com","password":"admin123"}'
```

### Products
- List available products
```curl
curl -X GET http://localhost:8085/products \
  -H "Authorization: Bearer <TOKEN>"
```

### Checkout
- Reserve product stock from a warehouse
- Reservation stored in Redis with expiration TTL
- Worker automatically releases stock when reservation expires
```curl
curl -X POST http://localhost:8085/checkout \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"product_id":1,"quantity":2}'
```

### Payment
- Mark order as paid
- Release reservation and update stock
- Update order status
```curl
curl -X POST http://localhost:8085/pay \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"order_id":1}'
```

### Stock Transfer
- Transfer stock between warehouses
- Ensures warehouse is active
- Transactional and consistent
```curl
curl -X POST http://localhost:8085/transfer \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"from_warehouse_id":1,"to_warehouse_id":2,"product_id":1,"quantity":5}'
```

### Warehouse
- Unified endpoint for activating/deactivating warehouse status
```curl
curl -X POST http://localhost:8085/warehouse/{id}/update-status \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"status":"active"}'
```

---

## How to use
- **make all** = Build and running application (http and worker)
- **make down** = Alias from docker compose down
- **make migrate** = Migrate schema and seeding data dummy
- **make rollback** = Drop all table