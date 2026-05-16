package config

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Config holds server configuration
type ServerConfig struct {
	// Server settings
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	Mode            string        `mapstructure:"mode"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes"`

	// CORS settings
	EnableCORS       bool     `mapstructure:"enable_cors"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"cors_max_age"`

	// Middleware settings
	EnableGzip      bool `mapstructure:"enable_gzip"`
	EnablePprof     bool `mapstructure:"enable_pprof"`
	EnableRequestID bool `mapstructure:"enable_request_id"`
	EnableRecovery  bool `mapstructure:"enable_recovery"`

	// Rate limiting
	EnableRateLimit bool `mapstructure:"enable_rate_limit"`
	RateLimit       int  `mapstructure:"rate_limit"`

	// Trusted proxies
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// Server wraps Gin engine with dependencies
type Server struct {
	Engine *gin.Engine
	Config *ServerConfig
	HTTP   *http.Server

	// Dependencies
	Redis     *redis.Client
	JwtConfig JwtConfig
	// Add other dependencies: DB, Logger, etc.
}

// NewServer creates a new server instance with all configurations
func NewServer(cfg *ServerConfig, deps *Dependencies) *Server {
	// Set Gin mode
	gin.SetMode(cfg.Mode)

	// Create Gin engine
	engine := gin.New()

	// Create server instance
	srv := &Server{
		Engine:    engine,
		Config:    cfg,
		Redis:     deps.RedisClient,
		JwtConfig: deps.JwtConfig,
	}

	// Setup middlewares
	srv.setupMiddlewares()

	// Setup HTTP server
	srv.HTTP = &http.Server{
		Addr:           fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:        engine,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	// Set trusted proxies
	if len(cfg.TrustedProxies) > 0 {
		engine.SetTrustedProxies(cfg.TrustedProxies)
	}

	return srv
}

// setupMiddlewares configures all middleware
func (s *Server) setupMiddlewares() {
	// Request ID middleware (should be first)
	if s.Config.EnableRequestID {
		s.Engine.Use(requestid.New())
	}

	// Recovery middleware (recover from panics)
	if s.Config.EnableRecovery {
		s.Engine.Use(gin.CustomRecovery(s.recoveryHandler))
	}

	// Logger middleware
	s.Engine.Use(s.loggerMiddleware())

	// CORS middleware
	if s.Config.EnableCORS {
		s.Engine.Use(cors.New(cors.Config{
			AllowOrigins:     s.Config.AllowedOrigins,
			AllowMethods:     s.Config.AllowedMethods,
			AllowHeaders:     s.Config.AllowedHeaders,
			ExposeHeaders:    s.Config.ExposeHeaders,
			AllowCredentials: s.Config.AllowCredentials,
			MaxAge:           time.Duration(s.Config.MaxAge) * time.Second,
		}))
	}

	// Gzip compression middleware
	if s.Config.EnableGzip {
		s.Engine.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	// Security headers middleware
	s.Engine.Use(s.securityHeadersMiddleware())

	// Rate limiting middleware
	if s.Config.EnableRateLimit {
		s.Engine.Use(s.rateLimitMiddleware())
	}

	// Timeout middleware
	s.Engine.Use(s.timeoutMiddleware(30 * time.Second))

	// Pprof endpoints (only for development/debugging)
	if s.Config.EnablePprof {
		pprof.Register(s.Engine)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := s.HTTP.Addr
	fmt.Printf("🚀 Server starting on %s\n", addr)
	fmt.Printf("📝 Mode: %s\n", s.Config.Mode)
	fmt.Printf("📝 Port: %s\n", s.Config.Port)

	if err := s.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("🛑 Shutting down server...")

	if err := s.HTTP.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("✓ Server stopped gracefully")
	return nil
}

// GetRouter returns the Gin engine for route registration
func (s *Server) GetRouter() *gin.Engine {
	return s.Engine
}

// RegisterRoutes registers all application routes
func (s *Server) RegisterRoutes(routeRegistrars ...RouteRegistrar) {
	for _, registrar := range routeRegistrars {
		registrar.RegisterRoutes(s.Engine, s)
	}
}

// RouteRegistrar interface for route registration
type RouteRegistrar interface {
	RegisterRoutes(router *gin.Engine, srv *Server)
}

// Dependencies holds all application dependencies
type Dependencies struct {
	RedisClient *redis.Client
	JwtConfig   JwtConfig
	// Add: DB, Logger, EventBus, etc.
}

// ===================================
// CUSTOM MIDDLEWARES
// ===================================

// loggerMiddleware creates custom logger middleware
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		// Get request ID
		requestID := c.GetString("X-Request-ID")
		if requestID == "" {
			requestID = requestid.Get(c)
		}

		fmt.Printf("[GIN] %s | %3d | %13v | %15s | %-7s %s %s\n",
			time.Now().Format("2006/01/02 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)

		// Store request ID in context for logging
		c.Set("request_id", requestID)
	}
}

// securityHeadersMiddleware adds security headers
func (s *Server) securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// rateLimitMiddleware implements rate limiting using Redis
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.Redis == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", clientIP)

		ctx := c.Request.Context()

		// Increment counter
		count, err := s.Redis.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		// Set expiry on first request
		if count == 1 {
			s.Redis.Expire(ctx, key, time.Minute)
		}

		// Check rate limit
		if count > int64(s.Config.RateLimit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", s.Config.RateLimit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", s.Config.RateLimit-int(count)))

		c.Next()
	}
}

// timeoutMiddleware adds request timeout
func (s *Server) timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lewati middleware ini jika request adalah protokol WebSocket
		if c.IsWebsocket() {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			return
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "request timeout",
			})
			c.Abort()
		}
	}
}

// recoveryHandler handles panic recovery
func (s *Server) recoveryHandler(c *gin.Context, err interface{}) {
	requestID := requestid.Get(c)

	fmt.Printf("[PANIC] RequestID: %s | Error: %v\n", requestID, err)

	c.JSON(http.StatusInternalServerError, gin.H{
		"error":      "internal server error",
		"request_id": requestID,
	})
}

// ===================================
// HELPER METHODS FOR HANDLERS
// ===================================

// GetRedis returns Redis client from server
func (s *Server) GetRedis() *redis.Client {
	return s.Redis
}

// GetRequestID gets request ID from context
func GetRequestID(c *gin.Context) string {
	return requestid.Get(c)
}

// GetContext returns request context (for Redis, DB operations)
func GetContext(c *gin.Context) context.Context {
	return c.Request.Context()
}
