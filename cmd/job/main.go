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
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("🚀 Starting Go-Chat Workers")
	log.Println("═══════════════════════════════════════════════════════════════")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	log.Printf("✅ Config loaded | Environment: %s", cfg.AppEnv)

	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ PostgreSQL connected")
	defer db.Close()

	rmq, err := config.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	log.Println("✅ RabbitMQ connected")
	defer rmq.Close()

	workerChannel, err := rmq.CreateChannel()
	if err != nil {
		log.Fatalf("❌ Failed to create worker channel: %v", err)
	}
	log.Println("✅ Worker channel created")
	defer workerChannel.Close()

	firebaseCredentialPath := "chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json"
	log.Printf("📱 Firebase credential path: %s", firebaseCredentialPath)
	log.Printf("📱 Firebase project: chat-appliaction-19fd5")

	app, err := config.InitFirebase(context.Background(), "chat-appliaction-19fd5", firebaseCredentialPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Firebase: %v", err)
	}
	log.Println("✅ Firebase initialized successfully")

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("❌ Failed to get FCM client: %v", err)
	}
	log.Println("✅ FCM client ready")

	// 5. Setup repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	firebaseRepo := repository.NewFirebaseRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	roomRepo := repository.NewRoomRepository(db)

	// 6. Create workers
	notificationWorker := worker.NewNotificationWorker(workerChannel, fcmClient, userRepo, messageRepo, firebaseRepo, memberRepo, roomRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. Start workers
	if err := notificationWorker.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start notification worker: %v", err)
	}

	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("✅ All workers started successfully!")
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("📬 Waiting for messages...")
	log.Println("🛑 Press Ctrl+C to shutdown")
	log.Println("═══════════════════════════════════════════════════════════════")

	// 8. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("🛑 Shutting down workers...")
	log.Println("═══════════════════════════════════════════════════════════════")
	cancel() // Stop all workers

	log.Println("✅ Workers exited gracefully")
	log.Println("═══════════════════════════════════════════════════════════════")
}
