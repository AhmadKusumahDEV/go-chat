package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/worker"
	"github.com/AhmadKusumahDEV/go-chat/pkg/httpclient"
	"github.com/AhmadKusumahDEV/go-chat/pkg/message-broker/rabbitmq"
)

func main() {
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("Starting Go-Chat Workers")
	log.Println("═══════════════════════════════════════════════════════════════")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	httpClient := httpclient.NewStdHTTPClient()

	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	log.Println("RabbitMQ connected")
	defer rmq.Close()

	// create worker channel
	workerChannel, err := rmq.CreateChannel()
	if err != nil {
		log.Fatalf("Failed to create worker channel: %v", err)
	}
	defer workerChannel.Close()

	firebaseCredentialPath := "chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json"

	app, err := config.InitFirebase(context.Background(), "chat-appliaction-19fd5", firebaseCredentialPath)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}
	log.Println("Firebase initialized successfully")

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Failed to get FCM client: %v", err)
	}
	log.Println("FCM client ready")

	// setup repository
	userRepo := repository.NewUserRepository(db)
	orderRepository := repository.NewOrderRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	firebaseRepo := repository.NewFirebaseRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	roomRepo := repository.NewRoomRepository(db)

	notificationWorker := worker.NewNotificationWorker(workerChannel, fcmClient, userRepo, messageRepo, firebaseRepo, memberRepo, roomRepo)
	CallNotificationWorker := worker.NewCallNotificationWorker(workerChannel, fcmClient, userRepo, messageRepo, firebaseRepo, memberRepo, roomRepo)
	emailWorker := worker.NewEmailWorker(workerChannel, httpClient, userRepo, orderRepository, cfg.Esp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := notificationWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start notification worker: %v", err)
	}

	if err := CallNotificationWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start call notification worker: %v", err)
	}

	if err := emailWorker.Start(ctx); err != nil {
		log.Printf("Failed to start EmailWorker: %v", err)
	}

	// 8. Wait for shutdown signal
	log.Println("waiting signal exit")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down Go-Chat Workers...")
}
