package controller

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/database"
)

type DBModuleConfig struct {
	team             *Team
	ModuleIdentifier marvin.ModuleID
	defaults         map[string]string
	protected        map[string]bool
	DefaultsLocked   bool
}

func newModuleConfig(t *Team, modID marvin.ModuleID) *DBModuleConfig {
	return &DBModuleConfig{
		team:             t,
		ModuleIdentifier: modID,
		defaults:         make(map[string]string),
		protected:        make(map[string]bool),
		DefaultsLocked:   false,
	}
}

func MigrateModuleConfig(c *database.Conn) error {
	err := c.Migrate("main", 1478022704,
		`CREATE TABLE config (
			id SERIAL PRIMARY KEY,
			module varchar(255),
			key varchar(255),
			value text,

			CONSTRAINT confkey UNIQUE(module, key)
		)`,
	)
	c.SyntaxCheck(
		sqlConfigGet,
		sqlConfigSet,
	)
	return err
}

const (
	sqlConfigGet = `SELECT value FROM config WHERE module = $1 AND key = $2`
	sqlConfigSet = `
		INSERT INTO config (module, key, value)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT confkey
		DO UPDATE SET value = excluded.value
			WHERE config.module = excluded.module
			AND config.key = excluded.key
	`
	sqlConfigReset = `
		DELETE FROM config
		WHERE module = $1 AND key = $2
	`
)

func (pc *DBModuleConfig) Add(key string, defaultValue string) {
	if pc.DefaultsLocked {
		panic("Module configuration must be set up during Load()")
	}
	pc.defaults[key] = defaultValue
}

func (pc *DBModuleConfig) AddProtect(key string, defaultValue string, protect bool) {
	if pc.DefaultsLocked {
		panic("Module configuration must be set up during Load()")
	}
	pc.defaults[key] = defaultValue
	pc.protected[key] = protect
}

func (pc *DBModuleConfig) Get(key string) (string, error) {
	def, haveDefault := pc.defaults[key]
	if !haveDefault {
		panic("Get() must have a default set")
	}

	stmt, err := pc.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(pc.ModuleIdentifier, key)

	var result sql.NullString
	err = row.Scan(&result)

	if !result.Valid {
		return def, nil
	} else if err != nil {
		return def, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	return result.String, nil
}

// GetIsDefault gets a module configuration value, but does not require the key have been initialized.
//
// 1) If the key was not initialized with Add(), value is the empty string, isDefault is true, and err is ErrConfNoDefault.
// 2) If the key was initialized, but has no override, value is the default value, isDefault is true, and err is nil.
// 3) If the key has an override, value is the override, isDefault is false, and err is nil.
//
// implements marvin.ModuleConfig.GetIsDefault
func (pc *DBModuleConfig) GetIsDefault(key string) (string, bool, error) {
	def, haveDefault := pc.defaults[key]

	stmt, err := pc.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(pc.ModuleIdentifier, key)
	var result sql.NullString
	err = row.Scan(&result)
	if !result.Valid {
		if haveDefault {
			return def, true, nil
		} else {
			return "", true, marvin.ErrConfNoDefault{Key: fmt.Sprintf("%s.%s", pc.ModuleIdentifier, key)}
		}
	} else if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	return result.String, false, nil
}

func (pc *DBModuleConfig) GetIsDefaultNotProtected(key string) (string, bool, error) {
	def, haveDefault := pc.defaults[key]

	if pc.protected[key] {
		return "__ERROR", true, marvin.ErrConfProtected{Key: fmt.Sprintf("%s.%s", pc.ModuleIdentifier, key)}
	}

	stmt, err := pc.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(pc.ModuleIdentifier, key)
	var result sql.NullString
	err = row.Scan(&result)
	if !result.Valid {
		if haveDefault {
			return def, true, nil
		} else {
			return "", true, marvin.ErrConfNoDefault{Key: fmt.Sprintf("%s.%s", pc.ModuleIdentifier, key)}
		}
	} else if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", pc.ModuleIdentifier, key)
	}
	return result.String, false, nil
}

func (pc *DBModuleConfig) Set(key, value string) error {
	stmt, err := pc.team.DB().Prepare(sqlConfigSet)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	defer stmt.Close()

	_, err = stmt.Exec(pc.ModuleIdentifier, key, value)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	return nil
}

func (pc *DBModuleConfig) SetDefault(key string) error {
	stmt, err := pc.team.DB().Prepare(sqlConfigReset)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	defer stmt.Close()

	_, err = stmt.Exec(pc.ModuleIdentifier, key)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", pc.ModuleIdentifier, key)
	}
	return nil
}

func (pc *DBModuleConfig) ListDefaults() map[string]string {
	if !pc.DefaultsLocked {
		//panic("ListDefaults() called before defaults locked")
	}
	return pc.defaults
}

func (pc *DBModuleConfig) ListProtected() map[string]bool {
	if !pc.DefaultsLocked {
		//panic("ListProtected() called before defaults locked")
	}
	return pc.protected
}
