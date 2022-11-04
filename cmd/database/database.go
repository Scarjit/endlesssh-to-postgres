package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"os"
)

var db *sql.DB

func OpenDatabase() *sql.DB {
	host, found := os.LookupEnv("DB_HOST")
	if !found {
		zap.S().Fatal("DB_HOST not set")
	}
	port, found := os.LookupEnv("DB_PORT")
	if !found {
		zap.S().Fatal("DB_PORT not set")
	}
	user, found := os.LookupEnv("DB_USER")
	if !found {
		zap.S().Fatal("DB_USER not set")
	}
	password, found := os.LookupEnv("DB_PASSWORD")
	if !found {
		zap.S().Fatal("DB_PASSWORD not set")
	}
	database, found := os.LookupEnv("DB_NAME")
	if !found {
		zap.S().Fatal("DB_NAME not set")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		database)
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		zap.S().Fatal(err)
	}
	return db
}

func CloseDatabase() {
	if db != nil {
		err := db.Close()
		if err != nil {
			zap.S().Fatal(err)
		}
	}
}

func InitDatabase() {
	// Create table ip_info
	var err error
	_, err = db.Exec(
		`
CREATE TABLE IF NOT EXISTS ip_info(
    id SERIAL PRIMARY KEY,
    ip text NOT NULL,
    hostname text NOT NULL ,
    anycast boolean NOT NULL,
    city text NOT NULL,
    region text NOT NULL,
    country text NOT NULL,
    loc text NOT NULL,
    org text NOT NULL,
    postal text NOT NULL,
    tz text NOT NULL
);`)
	if err != nil {
		zap.S().Fatal(err)
	}

	// Create connection table
	_, err = db.Exec(
		`
CREATE TABLE IF NOT EXISTS connections (
    date timestamptz NOT NULL,
    ip_info_id integer NOT NULL,
    duration interval NOT NULL,
    bytes bigint NOT NULL
);
`)
	if err != nil {
		zap.S().Fatal(err)
	}

	// Check if foreign key exists
	var exists bool
	err = db.QueryRow(
		`
SELECT EXISTS (
    SELECT 1
	FROM   information_schema.table_constraints
    	WHERE  constraint_type = 'FOREIGN KEY'
		AND    table_name = 'connections'
		AND    constraint_name = 'connections_ip_info_id_fkey'
);`).Scan(&exists)
	if err != nil {
		zap.S().Fatal(err)
	}

	if !exists {
		// Add foreign key constraint
		_, err = db.Exec(
			`
ALTER TABLE connections
ADD CONSTRAINT ip_info_id_fk
FOREIGN KEY (ip_info_id)
REFERENCES ip_info(id)
ON DELETE CASCADE;
`)
	}

	// Create hypertable
	_, err = db.Exec(`SELECT create_hypertable('connections', 'date', if_not_exists := TRUE);`)
	if err != nil {
		zap.S().Fatal(err)
	}

}
