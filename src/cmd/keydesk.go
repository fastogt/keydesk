package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"keydesk/app"
	"keydesk/app/version"

	console "log"

	"gopkg.in/yaml.v3"
)

type yamlConfig struct {
	Settings app.Config `yaml:"settings"`
}

func loadConfig(path string) (*app.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Settings.Host == "" {
		return nil, fmt.Errorf("settings.host is required")
	}
	if cfg.Settings.LogPath == "" {
		return nil, fmt.Errorf("settings.log_path is required")
	}
	if cfg.Settings.LogLevel == "" {
		return nil, fmt.Errorf("settings.log_level is required")
	}
	if cfg.Settings.Database == "" {
		return nil, fmt.Errorf("settings.database is required")
	}
	if cfg.Settings.JWTSecret == "" {
		return nil, fmt.Errorf("settings.jwt_secret is required")
	}
	if cfg.Settings.VaultMasterKey == "" {
		return nil, fmt.Errorf("settings.vault_master_key is required")
	}

	return &cfg.Settings, nil
}

func daemonize() {
	if err := os.Stdin.Close(); err != nil {
		console.Fatal(err)
	}
	if err := os.Stdout.Close(); err != nil {
		console.Fatal(err)
	}
	if err := os.Stderr.Close(); err != nil {
		console.Fatal(err)
	}
}

func pidfile() {
	if err := os.MkdirAll(version.RunDirPath, 0755); err != nil {
		console.Printf("Could not create run directory: %v", err)
	}
	pidFile, err := os.Create(version.PidFilePath)
	if err != nil {
		console.Printf("Could not create PID file: %v", err)
		return
	}
	fmt.Fprintf(pidFile, "%d", os.Getpid())
	pidFile.Close()
}

func removepid() {
	os.Remove(version.PidFilePath)
}

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "display version")

	var stopflag bool
	flag.BoolVar(&stopflag, "stop", false, "stop server")

	var daemonflag bool
	flag.BoolVar(&daemonflag, "daemon", false, "run server as daemon")

	var nopidflag bool
	flag.BoolVar(&nopidflag, "no-pid-file", false, "do not create pid file")

	var createAdmin string
	flag.StringVar(&createAdmin, "create-admin", "", "create admin user with given email")

	var adminPassword string
	flag.StringVar(&adminPassword, "password", "", "password for create-admin")

	configPath := flag.String("config", version.ConfigPath, "service config")

	flag.Parse()

	if ver {
		fmt.Printf("%s version %s\n", version.ProjectName, version.VersionApp)
		return
	}

	loadCfg := func() *app.Config {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			console.Fatalf("Failed to load config: %v", err)
		}
		return cfg
	}

	if stopflag {
		pidData, err := os.ReadFile(version.PidFilePath)
		if err != nil {
			console.Fatalf("Failed to read PID file: %v", err)
		}
		var pid int
		_, _ = fmt.Sscanf(string(pidData), "%d", &pid)
		process, err := os.FindProcess(pid)
		if err != nil {
			console.Fatalf("Failed to find process: %v", err)
		}
		if err := process.Signal(syscall.SIGTERM); err != nil {
			console.Fatalf("Failed to send signal: %v", err)
		}
		console.Printf("Stop signal sent to process %d", pid)
		return
	}

	if createAdmin != "" {
		if adminPassword == "" {
			console.Fatal("--password is required with --create-admin")
		}
		cfg := loadCfg()
		if err := app.CreateAdmin(cfg, createAdmin, adminPassword); err != nil {
			console.Fatalf("Failed to create admin: %v", err)
		}
		console.Printf("Admin user created: %s", createAdmin)
		return
	}

	cfg := loadCfg()

	a := app.NewApp()

	if daemonflag {
		daemonize()
	}

	if !nopidflag {
		pidfile()
	}

	var stopped int32
	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		sigsHub := make(chan os.Signal, 1)
		signal.Notify(sigsHub, syscall.SIGHUP)

		for {
			select {
			case sig := <-done:
				console.Printf("Received signal: %s", sig.String())
				atomic.StoreInt32(&stopped, 1)
				a.Stop()
			case <-sigsHub:
				console.Printf("Received reload config command")
				cfg = loadCfg()
				a.Stop()
			}
		}
	}()

	for atomic.LoadInt32(&stopped) != 1 {
		a.Initialize(*cfg)
		a.Run()
		a.DeInitialize()
	}

	if !nopidflag {
		removepid()
	}
}
