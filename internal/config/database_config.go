package config

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
)

type Database struct {
	*sqlx.DB
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

func NewDatabaseConnection(dbDriverName, connectionString string) (*Database, error) {
	database, err := sqlx.Connect(dbDriverName, connectionString)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	err = database.Ping()
	if err != nil {
		return nil, fmt.Errorf("ошибка пинга БД: %w", err)
	}

	log.Printf("подключение к БД выполнено успешно.\n Имя драйвера: %s \nСтрока подключения: %s", dbDriverName, connectionString)
	return &Database{database}, nil
}

func (d *Database) Close() error {
	err := d.DB.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия соединения с БД: %w", err)
	}
	return nil
}
