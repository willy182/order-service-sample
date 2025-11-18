package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"order-service-sample/helper"
	"order-service-sample/middleware"
	"order-service-sample/repository"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	db  *sql.DB
	rdb *redis.Client
)

func main() {
	var err error

	// === Load ENV ===
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://admin:nimda@db:5432/ecommerce?sslmode=disable"
	}
	redisAddr := helper.GetEnv("REDIS_ADDR", "redis:6379")

	// === Setup Postgres ===
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	// === Setup Redis ===
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	log.Println("connected to Redis:", redisAddr)

	// === Determine run mode ===
	mode := "app"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "worker":
		log.Println("Running in WORKER ONLY mode...")
		runWorker(context.Background())
		return

	case "app":
		log.Println("Running in HTTP SERVER mode only...")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		runHTTPServerWithShutdown(ctx, cancel)

	case "all":
		log.Println("Running in FULL mode (server + worker)...")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go runWorker(ctx)
		runHTTPServerWithShutdown(ctx, cancel)

	default:
		log.Fatalf("Unknown mode: %s (expected 'app', 'worker', or 'all')", mode)
	}
}

func runHTTPServer() {
	r := setupRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	log.Println("HTTP server running on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server error:", err)
	}
}

func runHTTPServerWithShutdown(ctx context.Context, cancel context.CancelFunc) {
	r := setupRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("HTTP server running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs
	log.Println("shutting down gracefully...")

	cancel()
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 5*time.Second)
	defer cancelTimeout()
	srv.Shutdown(ctxTimeout)
	log.Println("server and worker stopped.")
}

func setupRouter() *mux.Router {
	r := mux.NewRouter()

	// subrouter yang pakai middleware auth
	api := r.PathPrefix("/").Subrouter()
	api.Use(middleware.AuthMiddleware)

	api.HandleFunc("/products", ListProductsHandler).Methods("GET")
	api.HandleFunc("/checkout", CheckoutHandler).Methods("POST")
	api.HandleFunc("/pay", PayHandler).Methods("POST")
	api.HandleFunc("/transfer-product", TransferHandler).Methods("POST")
	api.HandleFunc("/warehouse/{id}/update-status", WarehouseUpdateStatusHandler).Methods("POST")

	// endpoint login tetap di luar auth
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	return r
}

func runWorker(ctx context.Context) {
	pubsub := rdb.PSubscribe(ctx, "__keyevent@0__:expired")
	defer pubsub.Close()

	log.Println("worker: listening to Redis expired events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: stopped.")
			return
		default:
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Println("pubsub receive error:", err)
				time.Sleep(time.Second)
				continue
			}

			key := msg.Payload
			if strings.HasPrefix(key, "reservation:") {
				idStr := strings.TrimPrefix(key, "reservation:")
				id, err := strconv.Atoi(idStr)
				if err != nil {
					log.Println("invalid order id in key:", idStr)
					continue
				}
				log.Printf("worker: reservation expired for order %d, releasing...\n", id)
				if err := repository.ReleaseReservationByOrderID(db, id); err != nil {
					log.Println("worker: failed to release reservation:", err)
				} else {
					log.Printf("worker: released reservation for order %d\n", id)
				}
			}
		}
	}
}
