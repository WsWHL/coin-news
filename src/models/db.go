package models

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"news/src/config"
	"time"
)

var DB *gorm.DB

func init() {
	opts := config.Cfg.Mysql

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opts.Username, opts.Password, opts.Host, opts.Port, opts.Database)
	sqlDb, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	sqlDb.SetMaxIdleConns(10)
	sqlDb.SetMaxOpenConns(50)
	sqlDb.SetConnMaxLifetime(time.Hour)

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                     sqlDb,
		DefaultStringSize:        64,
		DisableDatetimePrecision: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		panic(err)
	}
	db = db.Debug()

	// auto migrate
	_ = db.AutoMigrate(&Article{})

	DB = db
}
