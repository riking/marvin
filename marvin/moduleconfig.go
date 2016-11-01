package marvin

import (
	"database/sql"
	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin/database"
)

type DBModuleConfig struct {
	conn             *database.Conn
	ModuleIdentifier string
	defaults         map[string]string
}

func NewModuleConfig(c *database.Conn, moduleIdentifier string) ModuleConfig {
	return &DBModuleConfig{
		conn:             c,
		ModuleIdentifier: moduleIdentifier,
		defaults:         make(map[string]string),
	}
}

func MigrateModuleConfig(c *database.Conn) error {
	return c.Migrate("__core", 1478022704,
		`CREATE TABLE config (
			id SERIAL PRIMARY KEY,
			module varchar(255),
			key varchar(255),
			value text,

			CONSTRAINT confkey UNIQUE(module, key)
		)`,
	)
}

func (pc *DBModuleConfig) Add(key string, defaultValue string) {
	pc.defaults[key] = defaultValue
}

func (pc *DBModuleConfig) Get(key string) (string, error) {
	stmt, err := pc.conn.Prepare(`SELECT value FROM config WHERE module = $1 AND key = $2`)
	if err != nil {
		return "", errors.Wrapf(err, "moduleconfig.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	row := stmt.QueryRow(pc.ModuleIdentifier, key)
	var result sql.NullString
	err = row.Scan(&result)
	if !result.Valid {
		return pc.defaults[key], nil
	} else if err != nil {
		return "", errors.Wrapf(err, "moduleconfig.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	return result.String, nil
}

func (pc *DBModuleConfig) Set(key, value string) error {
	stmt, err := pc.conn.Prepare(`
		INSERT INTO config (module, key, value)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT confkey
		DO UPDATE SET value = excluded.value
			WHERE module = excluded.module
			AND key = excluded.key
	`)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	_, err = stmt.Exec(pc.ModuleIdentifier, key, value)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	return nil
}
