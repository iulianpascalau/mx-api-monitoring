package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/multiversx/mx-chain-core-go/core/check"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("api")

type server struct {
	router         *gin.Engine
	httpServer     *http.Server
	storage        Storage
	serviceKey     string
	username       string
	password       string
	listenAddr     string
	staticDir      string
	jwtSecret      []byte
	generalHandler func(http.Handler) http.Handler
	wg             sync.WaitGroup
}

// MetricReportPayload represents the incoming JSON body on /api/report
type MetricReportPayload struct {
	Metrics map[string]struct {
		Value          string `json:"value"`
		Type           string `json:"type"`
		NumAggregation int    `json:"numAggregation"`
	} `json:"metrics"`
}

// ArgsWebServer defines the web server arguments
type ArgsWebServer struct {
	ServiceKeyApi  string
	AuthUsername   string
	AuthPassword   string
	ListenAddress  string
	StaticDir      string
	Storage        Storage
	GeneralHandler func(http.Handler) http.Handler
}

// NewServer initializes the Gin engine and mounts all routes
func NewServer(args ArgsWebServer) (*server, error) {
	if check.IfNil(args.Storage) {
		return nil, errors.New("storage is required")
	}
	if args.GeneralHandler == nil {
		return nil, errors.New("nil http handler")
	}

	// Derive JWT secret from ServiceApiKey + random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	h := hmac.New(sha256.New, []byte(args.ServiceKeyApi))
	h.Write(salt)
	jwtSecret := h.Sum(nil)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())

	s := &server{
		router:         router,
		storage:        args.Storage,
		serviceKey:     args.ServiceKeyApi,
		username:       args.AuthUsername,
		password:       args.AuthPassword,
		listenAddr:     args.ListenAddress,
		staticDir:      args.StaticDir,
		generalHandler: args.GeneralHandler,
		jwtSecret:      jwtSecret,
	}

	s.setupRoutes()
	return s, nil
}

func (s *server) setupRoutes() {
	api := s.router.Group("/api")

	// Agent reporting endpoint
	api.POST("/report", s.authAPIKey(), s.handleReport)

	// Frontend authentication
	api.POST("/auth/login", s.handleLogin)

	// Protected frontend endpoints
	protected := api.Group("/")
	protected.Use(s.authJWT())
	{
		protected.GET("/metrics", s.handleGetMetrics)
		protected.GET("/metrics/:name/history", s.handleGetMetricHistory)
		protected.DELETE("/metrics/:name", s.handleDeleteMetric)
	}

	// Serve static files from the frontend build if configured
	if s.staticDir != "" {
		log.Info("serving static files", "dir", s.staticDir)
		s.router.Static("/static", path.Join(s.staticDir, "static"))
		s.router.StaticFile("/favicon.ico", path.Join(s.staticDir, "favicon.ico"))
		// Add other static assets if necessary

		// NoRoute for SPA fallback
		s.router.NoRoute(func(c *gin.Context) {
			// If request is for an /api route that doesn't exist, return 404
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "api route not found"})
				return
			}
			// Otherwise serve index.html for CSR
			c.File(path.Join(s.staticDir, "index.html"))
		})
	}
}

// Start listens and serves connections
func (s *server) Start() {
	handler := s.generalHandler(s.router)

	s.httpServer = &http.Server{
		Addr:    s.listenAddr,
		Handler: handler,
	}

	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		log.Error("failed to listen", "error", err)
		return
	}
	s.listenAddr = ln.Addr().String()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Info("starting HTTP server", "address", s.listenAddr)

		err := s.httpServer.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
		}
	}()
}

// Address returns the actual listen address
func (s *server) Address() string {
	return s.listenAddr
}

// Close gracefully stops the server
func (s *server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}
	s.wg.Wait()
	return s.storage.Close()
}

// --- Middlewares ---

func (s *server) authAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Api-Key")
		if key != s.serviceKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// VERY basic JWT implementation for frontend session based on HS256
func (s *server) authJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		parts := strings.Split(tokenStr, ".")
		if len(parts) != 3 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Verify signature
		message := parts[0] + "." + parts[1]
		sig, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token sign"})
			c.Abort()
			return
		}

		macd := hmac.New(sha256.New, s.jwtSecret)
		macd.Write([]byte(message))
		expectedSig := macd.Sum(nil)

		if !hmac.Equal(sig, expectedSig) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Verify expiration
		var claims struct {
			Exp int64 `json:"exp"`
		}
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			_ = json.Unmarshal(payloadBytes, &claims)
		}

		if time.Now().Unix() > claims.Exp {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// --- Handlers ---

func (s *server) handleReport(c *gin.Context) {
	var payload MetricReportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	recordedAt := time.Now().Unix()
	ctx := c.Request.Context()

	log.Debug("received report", "sender", c.Request.RemoteAddr, "num metrics", len(payload.Metrics))

	// In real-world, we could parallelize or bulk this, but for SQLite WAL, serial Tx is fine.
	for name, m := range payload.Metrics {
		err := s.storage.SaveMetric(ctx, name, m.Type, m.NumAggregation, m.Value, recordedAt)
		if err != nil {
			log.Warn("failed to save metric", "name", name, "error", err)
			// Continue with others
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if req.Username != s.username || req.Password != s.password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate basic JWT (Header.Payload.Signature)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := fmt.Sprintf(`{"sub":"%s","exp":%d}`, req.Username, time.Now().Add(24*time.Hour).Unix())
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))

	msg := header + "." + payload
	macd := hmac.New(sha256.New, s.jwtSecret)
	macd.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(macd.Sum(nil))

	token := msg + "." + sig
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *server) handleGetMetrics(c *gin.Context) {
	results, err := s.storage.GetLatestMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format to match specs.md exactly
	type responseMetric struct {
		Name           string `json:"name"`
		Value          string `json:"value"`
		Type           string `json:"type"`
		NumAggregation int    `json:"numAggregation"`
		RecordedAt     int64  `json:"recordedAt"`
	}

	out := make([]responseMetric, 0, len(results))
	for _, r := range results {
		if len(r.History) > 0 {
			out = append(out, responseMetric{
				Name:           r.Name,
				Value:          r.History[0].Value,
				Type:           r.Type,
				NumAggregation: r.NumAggregation,
				RecordedAt:     r.History[0].RecordedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"metrics": out})
}

func (s *server) handleGetMetricHistory(c *gin.Context) {
	name := c.Param("name")
	hist, err := s.storage.GetMetricHistory(c.Request.Context(), name)
	if err != nil {
		if err.Error() == "metric not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hist)
}

func (s *server) handleDeleteMetric(c *gin.Context) {
	name := c.Param("name")
	err := s.storage.DeleteMetric(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
