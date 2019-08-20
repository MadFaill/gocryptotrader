package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/database"
	db "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite"
	mg "github.com/thrasher-corp/gocryptotrader/database/migration"
)

var (
	dbConn          *database.Database
	configFile      string
	defaultDataDir  string
	createMigration string
	migrationDir    string
)

var defaultMigration = []byte(`-- up
-- down
`)

func openDbConnection(driver string) (err error) {
	if driver == "postgres" {
		dbConn, err = db.Connect()
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		dbConn.SQL.SetMaxOpenConns(2)
		dbConn.SQL.SetMaxIdleConns(1)
		dbConn.SQL.SetConnMaxLifetime(time.Hour)

	} else if driver == "sqlite" {
		dbConn, err = dbsqlite3.Connect()

		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}
	}
	return nil
}

type tmpLogger struct{}

// Printf implantation of migration Logger interface
// Passes directly to Printf from fmt package
func (t tmpLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

// Println implantation of migration Logger interface
// Passes directly to Println from fmt package
func (t tmpLogger) Println(v ...interface{}) {
	fmt.Println(v...)
}

// Errorf implantation of migration Logger interface
// Passes directly to Printf from fmt package
func (t tmpLogger) Errorf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func main() {
	fmt.Println("GoCryptoTrader database migration tool")
	fmt.Println(core.Copyright)
	fmt.Println()

	defaultPath, err := config.GetFilePath("")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	flag.StringVar(&configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&defaultDataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.StringVar(&createMigration, "create", "", "create a new empty migration file")
	flag.StringVar(&migrationDir, "migrationdir", mg.MigrationDir, "override migration folder")

	flag.Parse()

	if createMigration != "" {
		err = newMigrationFile(createMigration)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Migration created successfully")
		os.Exit(0)
	}

	tempLogger := tmpLogger{}

	temp := mg.Migrator{
		Log: tempLogger,
	}

	err = temp.LoadMigrations()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	conf := config.GetConfig()

	err = conf.LoadConfig(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	err = openDbConnection(conf.Database.Driver)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Connected to: %s\n", conf.Database.Host)

	temp.Conn = dbConn

	err = temp.RunMigration()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if dbConn.SQL != nil {
		err = dbConn.SQL.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func newMigrationFile(filename string) error {
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	path := filepath.Join(migrationDir, curTime+"_"+filename+".sql")
	err := common.CreateDir(migrationDir)
	if err != nil {
		return err
	}
	fmt.Printf("Creating new empty migration: %v\n", path)
	f, err := os.Create(path)

	if err != nil {
		return err
	}

	_, err = f.Write(defaultMigration)

	if err != nil {
		return err
	}

	return f.Close()
}
