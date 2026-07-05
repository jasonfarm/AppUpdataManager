package main

import (
	"example.com/appupdatemanager/server/config"
	"example.com/appupdatemanager/server/internal/api"
	"example.com/appupdatemanager/server/internal/middleware"
	"example.com/appupdatemanager/server/internal/store"
	"example.com/appupdatemanager/server/internal/ws"
	"flag"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// main 解析命令行参数，加载配置、初始化数据库，并启动 HTTP/WebSocket 服务。
func main() {
	var (
		addr       = flag.String("addr", "0.0.0.0:8080", "HTTP listen address")
		configPath = flag.String("config", "config/accounts.txt", "accounts config file path")
		dataDir    = flag.String("data", "./data", "data directory for uploads and database")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := store.Open(*dataDir)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := cfg.SyncUsers(db); err != nil {
		log.Fatalf("failed to sync users: %v", err)
	}

	hub := ws.NewHub(db)
	go hub.Run()

	r := gin.Default()

	// Static files for uploaded software/client binaries
	r.Static("/files", *dataDir+"/files")

	// WebSocket endpoint for clients
	r.GET("/ws", func(c *gin.Context) {
		ws.Serve(hub, c.Writer, c.Request)
	})

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/login", api.Login(cfg, db))
		apiGroup.GET("/me", middleware.Auth(cfg), api.Me)

		apiGroup.GET("/software", middleware.Auth(cfg), api.ListSoftware(db))
		apiGroup.POST("/software", middleware.Auth(cfg), api.CreateSoftware(db, *dataDir))
		apiGroup.DELETE("/software/:id", middleware.Auth(cfg), api.DeleteSoftware(db, *dataDir))
		apiGroup.POST("/software/:id/latest", middleware.Auth(cfg), api.SetLatestSoftware(db))
		apiGroup.PUT("/software/:id/name", middleware.Auth(cfg), api.UpdateSoftwareName(db))

		apiGroup.GET("/client-versions", middleware.Auth(cfg), api.ListClientVersions(db))
		apiGroup.POST("/client-versions", middleware.Auth(cfg), api.CreateClientVersion(db, *dataDir))
		apiGroup.POST("/client-versions/:id/latest", middleware.Auth(cfg), api.SetLatestClientVersion(db))

		apiGroup.GET("/resource-packages", middleware.Auth(cfg), api.ListResourcePackages(db))
		apiGroup.POST("/resource-packages", middleware.Auth(cfg), api.CreateResourcePackage(db, *dataDir))
		apiGroup.DELETE("/resource-packages/:id", middleware.Auth(cfg), api.DeleteResourcePackage(db, *dataDir))
		apiGroup.POST("/resource-packages/:id/latest", middleware.Auth(cfg), api.SetLatestResourcePackage(db))
		apiGroup.PUT("/resource-packages/:id/name", middleware.Auth(cfg), api.UpdateResourcePackageName(db))

		apiGroup.GET("/clients", middleware.Auth(cfg), api.ListClients(db))
		apiGroup.GET("/clients/:id", middleware.Auth(cfg), api.GetClient(db))
		apiGroup.GET("/clients/:id/commands", middleware.Auth(cfg), api.ListClientCommands(db))
		apiGroup.POST("/clients/:id/update-software", middleware.Auth(cfg), api.UpdateClientSoftware(hub, db))
		apiGroup.POST("/clients/:id/update-resource", middleware.Auth(cfg), api.UpdateClientResource(hub, db))
		apiGroup.POST("/clients/:id/update-self", middleware.Auth(cfg), api.UpdateClientSelf(hub, db))
		apiGroup.POST("/clients/:id/start", middleware.Auth(cfg), api.StartClientSoftware(hub, db))
		apiGroup.POST("/clients/:id/stop", middleware.Auth(cfg), api.StopClientSoftware(hub, db))
		apiGroup.POST("/clients/:id/restart", middleware.Auth(cfg), api.RestartClientSoftware(hub, db))
		apiGroup.PUT("/clients/:id/name", middleware.Auth(cfg), api.UpdateClientName(hub, db))
		apiGroup.DELETE("/clients/:id", middleware.Auth(cfg), api.DeleteClient(db))
	}

	// Serve frontend static files (production build)
	r.Static("/assets", "./static/assets")
	r.StaticFile("/", "./static/index.html")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" || path == "/index.html" {
			return
		}
		// For SPA routes, serve index.html unless it's an API/file/WebSocket path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/files") || path == "/ws" {
			c.AbortWithStatus(404)
			return
		}
		c.File("./static/index.html")
	})

	log.Printf("server listening on %s", *addr)
	if err := r.Run(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
