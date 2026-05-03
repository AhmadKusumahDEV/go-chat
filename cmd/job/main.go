// // cmd/worker/main.go
// package main

// import (
// 	"context"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"syscall"

// 	firebase "firebase.google.com/go/v4"
// 	"github.com/rabbitmq/amqp091-go"
// 	"google.golang.org/api/option"

// 	"myapp/internal/repository"
// 	"myapp/internal/worker"
// )

// func main() {
// 	log.Println("🔄 Starting workers...")

// 	// 1. Setup database
// 	db := setupDatabase()
// 	defer db.Close()

// 	// 2. Setup RabbitMQ
// 	rabbitmqConn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rabbitmqConn.Close()

// 	rabbitmqChannel, err := rabbitmqConn.Channel()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rabbitmqChannel.Close()

// 	// 3. Setup Firebase FCM
// 	opt := option.WithCredentialsFile("firebase-credentials.json")
// 	app, err := firebase.NewApp(context.Background(), nil, opt)
// 	if err != nil {
// 		log.Fatalf("Failed to initialize Firebase: %v", err)
// 	}

// 	fcmClient, err := app.Messaging(context.Background())
// 	if err != nil {
// 		log.Fatalf("Failed to get FCM client: %v", err)
// 	}

// 	// 4. Setup repositories
// 	messageRepo := repository.NewMessageRepository(db)
// 	userRepo := repository.NewUserRepository(db)

// 	// 5. Create workers
// 	messageWorker := worker.NewMessageWorker(rabbitmqChannel, messageRepo)
// 	notificationWorker := worker.NewNotificationWorker(rabbitmqChannel, fcmClient, userRepo, messageRepo)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	// 6. Start workers (RUNNING TERUS MENERUS!)
// 	if err := messageWorker.Start(ctx); err != nil {
// 		log.Fatalf("Failed to start message worker: %v", err)
// 	}

// 	if err := notificationWorker.Start(ctx); err != nil {
// 		log.Fatalf("Failed to start notification worker: %v", err)
// 	}

// 	log.Println("✅ All workers started")

// 	// 7. Wait for shutdown signal
// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
// 	<-quit

// 	log.Println("🛑 Shutting down workers...")
// 	cancel() // Stop all workers

// 	log.Println("✅ Workers exited");