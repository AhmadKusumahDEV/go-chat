package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/worker"
)

func main() {
	log.Println("🔄 Starting workers...")

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Setup database
	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 3. Setup RabbitMQ (using config wrapper with auto-reconnect)
	rmq, err := config.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	// Worker needs its own dedicated channel
	workerChannel, err := rmq.CreateChannel()
	if err != nil {
		log.Fatalf("Failed to create worker channel: %v", err)
	}
	defer workerChannel.Close()

	// 4. Setup Firebase FCM
	firebaseCredentialPath := "chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json"
	app, err := config.InitFirebase(context.Background(), "chat-appliaction-19fd5", firebaseCredentialPath)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Failed to get FCM client: %v", err)
	}

	// 5. Setup repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	firebaseRepo := repository.NewFirebaseRepository(db)
	memberRepo := repository.NewMemberRepository(db)

	// 6. Create workers
	notificationWorker := worker.NewNotificationWorker(workerChannel, fcmClient, userRepo, messageRepo, firebaseRepo, memberRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. Start workers
	if err := notificationWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start notification worker: %v", err)
	}

	log.Println("✅ All workers started")

	// 8. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down workers...")
	cancel() // Stop all workers

	log.Println("✅ Workers exited")
}