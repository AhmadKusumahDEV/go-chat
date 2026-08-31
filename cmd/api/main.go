package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"firebase.google.com/go/v4/messaging"
	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/router"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/AhmadKusumahDEV/go-chat/internal/worker"
	"github.com/AhmadKusumahDEV/go-chat/pkg/httpclient"
	"github.com/AhmadKusumahDEV/go-chat/pkg/message-broker/rabbitmq"
	"github.com/AhmadKusumahDEV/go-chat/pkg/redis"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
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

	s3Object, err := storage.NewMinioStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	httpClient := httpclient.NewStdHTTPClient()

	err = s3Object.Ping(appContext)
	log.Println(err)

	firebaseCredentialPath := "chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json"
	app, err := config.InitFirebase(appContext, "chat-appliaction-19fd5", firebaseCredentialPath)
	if err != nil {
		log.Printf("Failed to initialize Firebase: %v", err)
		log.Println("Push notification endpoints will not work")
	}

	var fcmClient *messaging.Client
	if app != nil {
		fcmClient, err = app.Messaging(appContext)
		if err != nil {
			log.Printf("Failed to get FCM client: %v", err)
		} else {
			log.Println("FCM client ready")
		}
	}

	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	rds, err := redis.NewClient(appContext, cfg.Redis.Addr, "")
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
	attacmentsRepository := repository.NewAttachmentsRepository(db)
	orderRepository := repository.NewOrderRepository(db)

	publisher := queue.NewRabbitMQPublisher(rmq.GetChannel())

	roomServices := services.NewRoomServices(roomRepository, memberRepository, attacmentsRepository, newClientRedis, validate, s3Object)
	usersServices := services.NewUsersServices(usersRepository, firebaseRepository, cfg.Jwt, cfg.Esp, s3Object, rds, httpClient, publisher)
	oauthServices := services.NewOauthServices(cfg, newClientRedis, oauthStatesRepository, usersRepository)
	messageServices := services.NewMessageServices(messageRepository, memberRepository, s3Object)
	memberServices := services.NewMemberServices(roomRepository, memberRepository, usersRepository, messageRepository, newClientRedis, validate)
	orderServices := services.NewOrderServices(cfg, usersRepository, httpClient, orderRepository, rds, publisher)

	manager := websocket.NewWebSocketManager(messageServices, roomServices, publisher, rmq.GetChannel(), usersServices)
	manager.Start()
	wsHandler := handlers.NewWebsocketHandler(manager, roomServices)
	wsRouter := router.NewWebsocketRouter(wsHandler)

	uploadworker := worker.NewDispatcher(s3Object, rds, cfg, attacmentsRepository, manager, 10, 1000)
	uploadworker.StartWorkerPool()

	defer manager.Stop()

	//handler
	roomHandler := handlers.NewRoomHandler(roomServices)
	UsersHandler := handlers.NewUserHandler(usersServices)
	oauthHandler := handlers.NewHandlerOauth(oauthServices)
	messageHandler := handlers.NewMessageHandler(messageServices, uploadworker, cfg, rds)
	memberHandler := handlers.NewMemberHandler(memberServices)
	testPushHandler := handlers.NewTestPushNotificationHandler(fcmClient, firebaseRepository, roomRepository)
	orderHandler := handlers.NewOrderHandler(orderServices)

	// router
	roomRouter := router.NewRoomRouter(roomHandler)
	UsersRouter := router.NewUsersRouter(UsersHandler)
	authRouter := router.NewAuthRouter(oauthHandler)
	messageRouter := router.NewMessageRouter(messageHandler)
	memberRouter := router.NewMemberRouter(memberHandler)
	orderRouter := router.NewOrderRouter(orderHandler)
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

	server.RegisterRoutes(roomRouter, UsersRouter, wsRouter, authRouter, messageRouter, memberRouter, testPushRouter, orderRouter)

	err = server.Start()

	if err != nil {
		panic(err)
	}

	err = server.Shutdown(appContext)

	if err != nil {
		panic(err)
	}

}
