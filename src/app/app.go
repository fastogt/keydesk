package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/fastogt/gofastogt/gofastogt"

	"keydesk/app/api"
	"keydesk/app/database"
	"keydesk/app/vault"
	"keydesk/app/version"
)

const logFileMaxSize = 10 * 1024 * 1024

type Config struct {
	Host           string `yaml:"host"`
	LogPath        string `yaml:"log_path"`
	LogLevel       string `yaml:"log_level"`
	Database       string `yaml:"database"`
	JWTSecret      string `yaml:"jwt_secret"`
	VaultMasterKey string `yaml:"vault_master_key"`
	HTTPS          *struct {
		Key  string `yaml:"key,omitempty"`
		Cert string `yaml:"cert,omitempty"`
	} `yaml:"https,omitempty"`
}

type App struct {
	logFile   *os.File
	http      *http.Server
	server    *api.Server
	db        *database.Database
	httpsCert string
	httpsKey  string
}

func NewApp() *App {
	return &App{}
}

func (app *App) Initialize(cfg Config) {
	logPath, err := gofastogt.StableFilePath(cfg.LogPath)
	if err != nil {
		log.Fatalf("Failed to resolve log path %s: %v", cfg.LogPath, err)
	}
	app.logFile, err = gofastogt.InitLogFile(*logPath, logFileMaxSize)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", *logPath, err)
	}
	log.SetOutput(app.logFile)

	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to parse log level %s: %v", cfg.LogLevel, err)
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(level)
	log.Infof("Running %s version %s", version.ProjectName, version.VersionApp)

	dbPath, err := gofastogt.StableFilePath(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to resolve database path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}
	app.db, err = database.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	log.Infof("Database: %s", *dbPath)

	v, err := vault.New(cfg.VaultMasterKey)
	if err != nil {
		log.Fatalf("Failed to initialize vault: %v", err)
	}
	log.Info("Vault initialized")

	app.server = api.NewServer(api.ServerConfig{
		Database:  app.db,
		JWTSecret: cfg.JWTSecret,
		Vault:     v,
	})

	app.http = &http.Server{
		Addr:    cfg.Host,
		Handler: app.server.Routes(),
	}
	if cfg.HTTPS != nil {
		app.httpsCert = cfg.HTTPS.Cert
		app.httpsKey = cfg.HTTPS.Key
	}

	log.Infof("Listening on %s", cfg.Host)
	log.Infof("Static files: %s", version.ShareFolderPath+"/install/public")
}

func (app *App) Run() {
	log.Info("Started http loop")
	var err error
	if app.httpsCert != "" && app.httpsKey != "" {
		log.Info("Starting HTTPS server")
		err = app.http.ListenAndServeTLS(app.httpsCert, app.httpsKey)
	} else {
		err = app.http.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		log.Errorf("http loop error: %s", err.Error())
	}
	log.Info("Finished http loop")
}

func (app *App) Stop() {
	log.Info("Stopping application...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if app.http != nil {
		if err := app.http.Shutdown(ctx); err != nil {
			log.Errorf("HTTP server shutdown failed: %v", err)
		}
		app.http = nil
	}

	if app.server != nil {
		app.server.Stop()
		app.server = nil
	}

	log.Info("Application stopped")
}

func (app *App) DeInitialize() {
	if app.db != nil {
		app.db.Close()
		app.db = nil
		log.Info("Database connection closed")
	}

	log.Infof("Quiting %s", version.VersionApp)

	if app.logFile != nil {
		app.logFile.Close()
		app.logFile = nil
	}
}

func CreateAdmin(cfg *Config, email, password string) error {
	dbPath, err := gofastogt.StableFilePath(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}
	db, err := database.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return db.CreateAdminUser(email, password)
}
