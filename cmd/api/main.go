package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"firebase.google.com/go/v4/messaging"
	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/router"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// var upgrade websocket.Upgrader = websocket.Upgrader{
// 	ReadBufferSize:  4096,
// 	WriteBufferSize: 4096,
// }

func main() {
	var err error

	appContext, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	defer func() {
		cancel()

		if err != nil {
			log.Fatal(err, "found error")
		}
	}()
	validate := validator.New()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Println("cannot load config:", err)
	}
	// log.Println(cfg.Firebase)

	// app, err := config.InitFirebase(appContext, "chat-appliaction-19fd5", cfg.Firebase.Path)
	// log.Println(err)
	// if err != nil {
	// 	log.Println("cannot init firebase:", err)
	// }

	// clientmessage, err := app.Messaging(appContext)
	// if err != nil {
	// 	log.Println("cannot setup message firebase:", err)
	// }

	// err = config.SendToDevice(appContext, clientmessage, "fH743pD6QnGKoMPv53JyOC:APA91bFTc8B8orxqrdsEDagmCHbrNDPyD27QUOinrhHAbuIecnhrO1jF5EFDRDCY5yYq6Bv-IpcN99ps6NzOg_RCVvB2JVF9rlEA29cBwasBB37fG4PtVFU")
	// if err != nil {
	// 	log.Println("cannot setup message firebase:", err)
	// }

	log.Println(cfg.DatabaseURL)
	log.Println(cfg.Redis)
	log.Println(cfg.RabbitMQ)

	// Initialize Firebase
	firebaseCredentialPath := "chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json"
	app, err := config.InitFirebase(appContext, "chat-appliaction-19fd5", firebaseCredentialPath)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Firebase: %v", err)
		log.Println("   Push notification endpoints will not work")
	}

	var fcmClient *messaging.Client
	if app != nil {
		fcmClient, err = app.Messaging(appContext)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to get FCM client: %v", err)
		} else {
			log.Println("✅ FCM client ready")
		}
	}

	// RabbitMQ (using config wrapper with auto-reconnect)
	rmq, err := config.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	rds, err := config.NewClient(appContext, cfg.Redis.Addr, "")
	if err != nil {
		panic(err)
	}

	newClientRedis := cahce.NewClientRedis(rds)

	// 1. Initialize Repositories
	roomRepository := repository.NewRoomRepository(db)
	memberRepository := repository.NewMemberRepository(db)
	usersRepository := repository.NewUserRepository(db)
	oauthStatesRepository := repository.NewOauthStatesRepository(db)
	messageRepository := repository.NewMessageRepository(db)
	firebaseRepository := repository.NewFirebaseRepository(db)

	// 2. Initialize Queue Publisher
	publisher := queue.NewRabbitMQPublisher(rmq.GetChannel())

	// 3. Initialize Services
	roomServices := services.NewRoomServices(roomRepository, memberRepository, newClientRedis, validate)
	usersServices := services.NewUsersServices(usersRepository, firebaseRepository, cfg.Jwt)
	oauthServices := services.NewOauthServices(cfg, newClientRedis, oauthStatesRepository, usersRepository)
	messageServices := services.NewMessageServices(messageRepository, memberRepository)
	memberServices := services.NewMemberServices(roomRepository, memberRepository, newClientRedis, validate)

	// 4. Initialize WebSocket Manager
	manager := websocket.NewWebSocketManager(messageServices, publisher)
	manager.Start()
	wsHandler := handlers.NewWebsocketHandler(manager)
	wsRouter := router.NewWebsocketRouter(wsHandler)

	defer manager.Stop()

	//handler
	roomHandler := handlers.NewRoomHandler(roomServices)
	UsersHandler := handlers.NewUserHandler(usersServices)
	oauthHandler := handlers.NewHandlerOauth(oauthServices)
	messageHandler := handlers.NewMessageHandler(messageServices)
	memberHandler := handlers.NewMemberHandler(memberServices)

	// router
	roomRouter := router.NewRoomRouter(roomHandler)
	UsersRouter := router.NewUsersRouter(UsersHandler)
	authRouter := router.NewAuthRouter(oauthHandler)
	messageRouter := router.NewMessageRouter(messageHandler)
	memberRouter := router.NewMemberRouter(memberHandler)
	testPushHandler := handlers.NewTestPushNotificationHandler(fcmClient, firebaseRepository, roomRepository)
	testPushRouter := handlers.NewTestPushRouter(testPushHandler)

	if err != nil {
		panic(err)
	}

	server := config.NewServer(&cfg.Server, &config.Dependencies{RedisClient: rds, JwtConfig: cfg.Jwt})

	server.Engine.GET("/", func(c *gin.Context) {
		c.JSON(200, struct {
			Nama   string
			Umur   int
			Posisi string
		}{
			Nama:   "Budi",
			Umur:   28,
			Posisi: "Developer",
		})
	})

	server.RegisterRoutes(roomRouter, UsersRouter, wsRouter, authRouter, messageRouter, memberRouter, testPushRouter)

	err = server.Start()

	if err != nil {
		panic(err)
	}

	err = server.Shutdown(appContext)

	if err != nil {
		panic(err)
	}

}
