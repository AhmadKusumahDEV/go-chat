package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/router"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rabbitmq/amqp091-go"
	// "github.com/AhmadKusumahDEV/go-chat/internal/config"
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

	log.Println(cfg.Jwt)
	log.Println(cfg.OAuth)
	log.Println(cfg.Server)
	log.Println(cfg.DatabaseURL)
	log.Println(cfg.Redis)
	log.Println(cfg.RabbitMQ)

	rabbitmqURL := cfg.RabbitMQ.URL
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}
	rabbitmqConn, err := amqp091.Dial(rabbitmqURL)
	if err != nil {
		log.Println("error in rabbitmq", err)
	}
	defer rabbitmqConn.Close()

	db, err := config.NewDB(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	rds, err := config.NewClient(appContext, cfg.Redis.Addr, "")
	if err != nil {
		panic(err)
	}

	newClientRedis := cahce.NewClientRedis(rds)

	// repostiory
	roomRepository := repository.NewRoomRepository(db)
	memberRepository := repository.NewMemberRepository(db)
	usersRepository := repository.NewUserRepository(db)
	oauthStatesRepository := repository.NewOauthStatesRepository(db)
	messageRepository := repository.NewMessageRepository(db)

	//services
	roomServices := services.NewRoomServices(roomRepository, memberRepository, newClientRedis, validate)
	usersServices := services.NewUsersServices(usersRepository, cfg.Jwt)
	oauthServices := services.NewOauthServices(cfg, newClientRedis, oauthStatesRepository, usersRepository)
	messageServices := services.NewMessageServices(messageRepository, memberRepository)

	//handler
	roomHandler := handlers.NewRoomHandler(roomServices)
	UsersHandler := handlers.NewUserHandler(usersServices)
	oauthHandler := handlers.NewHandlerOauth(oauthServices)
	messageHandler := handlers.NewMessageHandler(messageServices)

	// router
	roomRouter := router.NewRoomRouter(roomHandler)
	UsersRouter := router.NewUsersRouter(UsersHandler)
	authRouter := router.NewAuthRouter(oauthHandler)
	messageRouter := router.NewMessageRouter(messageHandler)

	if err != nil {
		panic(err)
	}

	manager := websocket.NewWebSocketManager(roomRepository)
	manager.Start()

	defer manager.Stop()

	wsHandler := handlers.NewWebsocketHandler(manager)
	wsRouter := router.NewWebsocketRouter(wsHandler)

	server := config.NewServer(&cfg.Server, &config.Dependencies{RedisClient: rds, JwtConfig: cfg.Jwt})

	server.Engine.Use(middelware.GinContextErrorHandler())

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

	server.RegisterRoutes(roomRouter, UsersRouter, wsRouter, authRouter, messageRouter)

	err = server.Start()

	if err != nil {
		panic(err)
	}

	err = server.Shutdown(appContext)

	if err != nil {
		panic(err)
	}

}
