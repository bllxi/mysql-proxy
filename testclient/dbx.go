package testclient

import (
	"database/sql"
	"fmt"

	"log"

	"github.com/bllxi/mysql-proxy/config"
	_ "github.com/go-sql-driver/mysql"
)

type DBX struct {
	db *sql.DB
}

func (d *DBX) Open() error {
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v",
		config.User, config.Password, config.ConnectAddr, config.DB)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	d.db = db

	log.Printf("open database %v:%v@tcp(%v)/%v ok!", config.SqlUser, config.SqlPwd, config.SqlAddr, config.SqlDB)

	return nil
}

func (d *DBX) Close() {
	d.db.Close()
}

func (d *DBX) Exec(sql string) (sql.Result, error) {
	return d.db.Exec(sql)
}

func (d *DBX) Query(sql string) (*sql.Rows, error) {
	return d.db.Query(sql)
}
