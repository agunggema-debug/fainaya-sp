package main

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	initDB()
	defer db.Close()

	e := echo.New()
	e.HideBanner = true

	// Standard middlewares
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} [${method}] ${uri} -> ${status} (${latency_human})\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	// API Routes Group
	api := e.Group("/api")

	// Public routes
	api.POST("/login", handleLogin)

	// Protected routes (Requires Auth Token / Cookie)
	auth := api.Group("", authMiddleware)
	auth.GET("/me", handleGetMe)
	auth.PATCH("/theme", handleUpdateTheme)

	// Tickets
	auth.GET("/tickets", handleGetTickets)
	auth.POST("/tickets", handleCreateTicket) // All logged-in can create
	auth.PATCH("/tickets/:id/status", handleUpdateTicketStatus, requireRoles("it_support", "store_manager", "ho_it", "vendor"))
	auth.POST("/tickets/:id/escalate", handleEscalateTicket, requireRoles("it_support", "store_manager", "ho_it"))

	// Checklists
	auth.GET("/checklists", handleGetChecklist)
	auth.POST("/checklists/:id/toggle", handleToggleChecklist, requireRoles("it_support", "super_admin"))

	// Assets
	auth.GET("/assets", handleGetAssets)

	// Software & Antivirus
	auth.GET("/software", handleGetSoftware)
	auth.GET("/antivirus", handleGetAntivirus)

	// Network
	auth.GET("/network/devices", handleGetNetworkDevices)
	auth.POST("/network/diagnostics", handleRunDiagnostic, requireRoles("it_support", "ho_it"))

	// Windows Server
	auth.GET("/server/health", handleGetServerHealth)
	auth.GET("/server/services", handleGetWindowsServices)
	auth.POST("/server/services/control", handleControlWindowsService, requireRoles("it_support", "ho_it"))

	// Logs & Knowledge Base & Vendors
	auth.GET("/logs", handleGetActivityLogs, requireRoles("it_support", "store_manager", "ho_it"))
	auth.GET("/knowledge-base", handleGetKnowledgeBase)
	auth.GET("/vendors", handleGetVendors)

	// User management
	auth.GET("/users", handleGetUsers, requireRoles("super_admin"))

	// Explicitly register standard MIME types
	_ = mime.AddExtensionType(".js", "application/javascript")
	_ = mime.AddExtensionType(".mjs", "application/javascript")
	_ = mime.AddExtensionType(".css", "text/css")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")

	// Determine absolute or reliable relative path for client/dist
	distPath := "../client/dist"
	if _, err := os.Stat(distPath); err != nil {
		if _, err := os.Stat("client/dist"); err == nil {
			distPath = "client/dist"
		}
	}

	if _, err := os.Stat(distPath); err == nil {
		// Serve all static assets from dist
		e.Static("/assets", filepath.Join(distPath, "assets"))
		e.File("/favicon.svg", filepath.Join(distPath, "favicon.svg"))
		e.File("/favicon.ico", filepath.Join(distPath, "favicon.svg"))

		// SPA Fallback for non-API routes
		e.GET("/*", func(c echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api") {
				return echo.ErrNotFound
			}
			cleanPath := filepath.Clean(filepath.Join(distPath, path))
			if fi, err := os.Stat(cleanPath); err == nil && !fi.IsDir() {
				return c.File(cleanPath)
			}
			return c.File(filepath.Join(distPath, "index.html"))
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Fainaya Store Support Server starting on port :%s ...", port)
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
