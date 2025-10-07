package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"alfredoptarigan/cv-evaluator/internal/config"
	"alfredoptarigan/cv-evaluator/internal/handlers"
	"alfredoptarigan/cv-evaluator/internal/repositories"
	"alfredoptarigan/cv-evaluator/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Println("✅ Config loaded successfully")

	// Initialize database
	db, err := config.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	// Initializes repositories
	docRepo := repositories.NewDocumentRepository(db)
	evalRepo := repositories.NewEvaluationRepository(db)
	log.Println("✅ Repositories initialized successfully")

	// Initialize services
	storageService := services.NewStorageService(cfg.Storage.UploadPath)
	if err := storageService.EnsureUploadDir(); err != nil {
		log.Fatalf("❌ Failed to create upload directory: %v", err)
	}

	pdfParser := services.NewPDFParserService()
	log.Println("✅ Services initialized successfully")

	// Initialize Gemini AI
	geminiService, err := services.NewGeminiService(cfg.Gemini.APIKey)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Gemini AI: %v", err)
	}
	log.Println("✅ Gemini AI initialized successfully")

	// Initialize Qdrant
	qdrantService, err := services.NewQdrantService(
		cfg.Qdrant.URL,
		cfg.Qdrant.APIKey,
		cfg.Qdrant.Collection,
	)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Qdrant: %v", err)
	}

	if err := qdrantService.InitCollection(); err != nil {
		log.Fatalf("❌ Failed to initialize Qdrant collection: %v", err)
	}
	log.Println("✅ Qdrant initialized successfully")

	// Initialize evaluator
	evaluatorService := services.NewEvaluatorService(
		evalRepo,
		docRepo,
		geminiService,
		qdrantService,
		pdfParser,
		cfg.Worker.RetryMaxAttempts,
	)
	log.Println("✅ Evaluator service initialized")

	// Initialize worker
	worker := services.NewWorker(
		evalRepo,
		evaluatorService,
		cfg.Worker.Concurrency,
	)
	log.Println("✅ Worker initialized successfully")

	// Start worker
	ctx := context.Background()
	worker.Start(ctx)
	log.Println("✅ Worker started successfully")

	// Initialize Handlers
	uploadHandler := handlers.NewUploadHandler(
		docRepo,
		storageService,
		cfg.Storage.MaxFileSize,
	)
	evaluateHandler := handlers.NewEvaluationHandler(
		evalRepo,
		docRepo,
		worker,
	)

	resultHandler := handlers.NewResultHandler(evalRepo)
	log.Println("✅ Handlers initialized")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "AI CV Evaluator API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		BodyLimit:    int(cfg.Storage.MaxFileSize),
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now(),
		})
	})

	// API endpoints
	api.Post("/upload", uploadHandler.HandleUpload)
	api.Post("/evaluate", evaluateHandler.HandleEvaluate)
	api.Get("/result/:id", resultHandler.HandleGetResult)

	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "AI CV Evaluator API",
			"version": "1.0.0",
			"endpoints": []string{
				"POST /api/v1/upload",
				"POST /api/v1/evaluate",
				"GET /api/v1/result/:id",
			},
		})
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("\n🛑 Shutting down server...")
		worker.Stop()
		if err := app.Shutdown(); err != nil {
			log.Printf("❌ Server forced to shutdown: %v", err)
		}
	}()

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("🚀 Server starting on %s\n", addr)
	log.Printf("📖 API Documentation: http://localhost%s\n", addr)

	if err := app.Listen(addr); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}

}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"code":  code,
	})
}
