package db

import (
	"fmt"
	"io/ioutil"
	"time"

	"gorm.io/driver/mysql"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

/* default db configs */
const (
	DefaultDBType          = "mysql"
	DefaultDBHost          = "localhost"
	DefaultDBPort          = "3306"
	DefaultMaxIdleConns    = 20
	DefaultMaxOpenConns    = 0
	DefaultConnMaxLifetime = 0
)

// Config hold all config of db
type Config struct {
	DBType          string            `yaml:"dbType,omitempty" description:"database type"`
	Username        string            `yaml:"username,omitempty" description:"database username"`
	Password        string            `yaml:"password,omitempty" description:"database password"`
	Host            string            `yaml:"host,omitempty" description:"database server host"`
	Port            string            `yaml:"port,omitempty" description:"database server port"`
	Schema          string            `yaml:"schema,omitempty" description:"database schema"`
	Params          map[string]string `yaml:"params,omitempty" description:"database schema params"`
	AutoMigrate     bool              `yaml:"autoMigrate,omitempty" description:"if auto migrate tables"`
	LogMode         int               `yaml:"logMode,omitempty" description:"database log mode, 1->Silent, 2->Error, 3->Warn, 4->Info"`
	MaxIdleConns    int               `yaml:"maxIdleConns,omitempty" description:"the maximum number of connections in the idle connection pool"`
	MaxOpenConns    int               `yaml:"maxOpenConns,omitempty" description:"the maximum number of open connections to the database"`
	ConnMaxLifetime time.Duration     `yaml:"connMaxLifetime,omitempty" description:"the max life time of a connection"`
}

// newConfig build default db config
func newConfig() *Config {
	return &Config{
		DBType:          DefaultDBType,
		Host:            DefaultDBHost,
		Port:            DefaultDBPort,
		MaxIdleConns:    DefaultMaxIdleConns,
		MaxOpenConns:    DefaultMaxOpenConns,
		ConnMaxLifetime: DefaultConnMaxLifetime,
	}
}

// buildDBAddr gen db address to connect
func (c *Config) buildDBAddr() string {
	addr := fmt.Sprintf("%s:%s@(%s:%s)/%s", c.Username, c.Password, c.Host, c.Port, c.Schema)
	if len(c.Params) > 0 {
		params := "?"
		for param, value := range c.Params {
			params += fmt.Sprintf("%s=%s&", param, value)
		}
		addr += params
	}
	return addr
}

// open connect to db server
func (c *Config) open() (*gorm.DB, error) {
	alog.Infof("Connecting to %s db", c.DBType)

	conn, err := gorm.Open(mysql.Open(c.buildDBAddr()), &gorm.Config{})
	if err != nil {
		alog.Errorf("Open db connect failed: %v", err)
		return nil, err
	}

	alog.Infof("Connect %s db succeed", c.DBType)
	return conn, err
}

// loadConfigFromFile load db config from a file
func loadConfigFromFile(filename string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		alog.Errorf("Read db config file failed: %v", err)
		return nil, err
	}
	config := newConfig()
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		alog.Errorf("DB config load failed: %v", err)
		return config, err
	}
	alog.V(4).Infof("DB config loaded: %s", filename)
	return config, nil
}
