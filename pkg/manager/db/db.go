package db

import (
	"sync"

	"gorm.io/gorm/logger"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"gorm.io/gorm"

	// registry mysql type
	_ "gorm.io/driver/mysql"
)

var singleton *DB
var once sync.Once
var dbTables []interface{}
var dbData []interface{}

// DB operator of database
type DB struct {
	config *Config
	db     *gorm.DB
}

// InitDB build a DB instance once
func InitDB(configfile string) *DB {
	// only one db instance existed
	once.Do(func() {
		config, err := loadConfigFromFile(configfile)
		if err != nil {
			alog.Fatalf("Load db config file failed: %v", err)
		}
		conn, err := config.open()
		if err != nil {
			alog.Fatalf("Open db connection failed: %v", err)
		}
		alog.Infof("Connect DB: %v", config.Host)

		db, err := conn.DB()
		if err != nil {
			alog.Fatalf("Open db failed: %v", err)
		}

		// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
		db.SetMaxIdleConns(20)

		// SetMaxOpenConns sets the maximum number of open connections to the database. Here no limit.
		db.SetMaxOpenConns(0)
		// SetConnMaxLifetime sets the maximum amount of time a connection may be reused. Here connections are reused forever.
		db.SetConnMaxLifetime(0)

		// auto migrate db
		if config.AutoMigrate {
			if err := conn.AutoMigrate(dbTables...); err != nil {
				alog.Fatalf("Auto migrate tables failed: %v", err)
			}
			//alog.Infof("Auto migrated db tables and data")
			initData(conn)
		}

		if config.LogMode > 0 {
			conn.Logger = logger.Default.LogMode(logger.LogLevel(config.LogMode))
		}
		singleton = &DB{
			config: config,
			db:     conn,
		}
	})
	return singleton
}

// initData save init data if needed
// FIXME remove if possible or load from csv file
func initData(db *gorm.DB) {
	// save some init data
	for _, d := range dbData {
		saveData(db, d)
	}
	// clear init data after init
	dbData = dbData[0:0]
}

// Get get gorm DB at anywhere
func Get() *gorm.DB {
	return singleton.db
}

// RegisterDBTable register table to create
func RegisterDBTable(table interface{}) {
	dbTables = append(dbTables, table)
}

func saveData(db *gorm.DB, data interface{}) {
	if err := db.Where(data).First(data).Error; err == gorm.ErrRecordNotFound {
		db.Save(data)
	}
}

// AddData add data for initializing
func AddData(data ...interface{}) {
	dbData = append(dbData, data...)
}

// NewTransaction get gorm DB tx begin at anywhere
func NewTransaction() *gorm.DB {
	return singleton.db.Begin()
}
