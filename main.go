package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"scoring_api_gateway/graph"
	"scoring_api_gateway/graph/model"
	"scoring_api_gateway/internal/config"
	"scoring_api_gateway/internal/logger"
	"scoring_api_gateway/internal/messaging"
	"scoring_api_gateway/internal/repository"
	"scoring_api_gateway/internal/service"
)

func runMigrations(db *pgxpool.Pool, log *zap.Logger) error {
	log.Info("Running database migrations")

	migrationsDir := "migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)

	for _, filename := range migrationFiles {
		log.Info("Running migration", zap.String("file", filename))

		content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		_, err = db.Exec(context.Background(), string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		log.Info("Migration completed", zap.String("file", filename))
	}

	log.Info("All migrations completed successfully")
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.JSON)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting scoring API gateway")

	db, err := pgxpool.New(context.Background(), cfg.DatabaseDSN())
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	log.Info("Connected to database")

	if err := runMigrations(db, log); err != nil {
		log.Fatal("Failed to run migrations", zap.Error(err))
	}

	natsClient, err := messaging.NewNATSClient(cfg.NATS.URL, log)
	if err != nil {
		log.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsClient.Close()

	log.Info("Connected to NATS")

	cacheRepo := repository.NewDataCacheRepository(db, log)
	verificationRepo := repository.NewVerificationRepository(db, cacheRepo, log)
	verificationService := service.NewVerificationService(verificationRepo, natsClient, log)

	// Подписываемся на уведомления о завершении обработки
	err = natsClient.SubscribeToVerificationCompleted(context.Background(), func(verification *model.Verification) {
		log.Info("Received verification completed notification",
			zap.String("verification_id", verification.ID),
			zap.String("status", string(verification.Status)))
	})
	if err != nil {
		log.Error("Failed to subscribe to verification completed", zap.Error(err))
	}

	// Внедряем зависимости в резолверы
	resolver := &graph.Resolver{
		VerificationService: verificationService,
		Logger:              log,
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	srv := handler.NewDefaultServer(schema)

	http.Handle("/query", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("GraphQL request received",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("user_agent", r.UserAgent()),
			zap.String("remote_addr", r.RemoteAddr))
		srv.ServeHTTP(w, r)
	}))

	http.Handle("/playground", playground.Handler("GraphQL playground", "/query"))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info("Starting server", zap.String("address", addr))

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	server := &http.Server{
		Addr: addr,
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited")
}
