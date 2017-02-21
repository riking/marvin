package controller

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/database"
)

type DBModuleConfig struct {
	team             *Team
	ModuleIdentifier marvin.ModuleID

	// All writes to the maps must happen during the Load() phase.
	// DefaultsLocked is set afterwards, in single-thread code.
	DefaultsLocked bool
	defaults       map[string]string
	protected      map[string]bool
	callbacks      []func(string)
}

func newModuleConfig(t *Team, modID marvin.ModuleID) marvin.ModuleConfig {
	c := &DBModuleConfig{
		team:             t,
		ModuleIdentifier: modID,

		DefaultsLocked: false,
		defaults:       make(map[string]string),
		protected:      make(map[string]bool),
		callbacks:      nil,
	}
	if modID == "blacklist" || modID == "apikeys" {
		return AllProtectedModuleConfig{c}
	}
	return c
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

func (c *DBModuleConfig) Add(key string, defaultValue string) {
	if c.DefaultsLocked {
		panic("Module configuration must be set up during Load()")
	}
	c.defaults[key] = defaultValue
}

func (c *DBModuleConfig) AddProtect(key string, defaultValue string, protect bool) {
	if c.DefaultsLocked {
		panic("Module configuration must be set up during Load()")
	}
	c.defaults[key] = defaultValue
	c.protected[key] = protect
}

func (c *DBModuleConfig) OnModify(f func(key string)) {
	if c.DefaultsLocked {
		panic("Module configuration must be set up during Load()")
	}
	c.callbacks = append(c.callbacks, f)
}

func (c *DBModuleConfig) Get(key string) (string, error) {
	def, haveDefault := c.defaults[key]
	if !haveDefault {
		panic("Get() must have a default set")
	}

	stmt, err := c.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(c.ModuleIdentifier, key)

	var result sql.NullString
	err = row.Scan(&result)

	if !result.Valid {
		return def, nil
	} else if err != nil {
		return def, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
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
func (c *DBModuleConfig) GetIsDefault(key string) (string, bool, error) {
	def, haveDefault := c.defaults[key]

	stmt, err := c.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(c.ModuleIdentifier, key)
	var result sql.NullString
	err = row.Scan(&result)
	if !result.Valid {
		if haveDefault {
			return def, true, nil
		} else {
			return "", true, marvin.ErrConfNoDefault{Key: fmt.Sprintf("%s.%s", c.ModuleIdentifier, key)}
		}
	} else if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
	}
	return result.String, false, nil
}

func (c *DBModuleConfig) GetIsDefaultNotProtected(key string) (string, bool, error) {
	def, haveDefault := c.defaults[key]

	if c.protected[key] {
		return "__ERROR", true, marvin.ErrConfProtected{Key: fmt.Sprintf("%s.%s", c.ModuleIdentifier, key)}
	}

	stmt, err := c.team.DB().Prepare(sqlConfigGet)
	if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
	}
	defer stmt.Close()

	row := stmt.QueryRow(c.ModuleIdentifier, key)
	var result sql.NullString
	err = row.Scan(&result)
	if !result.Valid {
		if haveDefault {
			return def, true, nil
		} else {
			return "", true, marvin.ErrConfNoDefault{Key: fmt.Sprintf("%s.%s", c.ModuleIdentifier, key)}
		}
	} else if err != nil {
		return def, true, errors.Wrapf(err, "config.get(%s, %s)", c.ModuleIdentifier, key)
	}
	return result.String, false, nil
}

func (c *DBModuleConfig) Set(key, value string) error {
	stmt, err := c.team.DB().Prepare(sqlConfigSet)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", c.ModuleIdentifier, key)
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.ModuleIdentifier, key, value)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", c.ModuleIdentifier, key)
	}

	for _, v := range c.callbacks {
		go v(key)
	}
	return nil
}

func (c *DBModuleConfig) SetDefault(key string) error {
	stmt, err := c.team.DB().Prepare(sqlConfigReset)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", c.ModuleIdentifier, key)
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.ModuleIdentifier, key)
	if err != nil {
		return errors.Wrapf(err, "moduleconfig.set(%s, %s)", c.ModuleIdentifier, key)
	}
	return nil
}

func (c *DBModuleConfig) ListDefaults() map[string]string {
	if !c.DefaultsLocked {
		//panic("ListDefaults() called before defaults locked")
	}
	return c.defaults
}

func (c *DBModuleConfig) ListProtected() map[string]bool {
	if !c.DefaultsLocked {
		//panic("ListProtected() called before defaults locked")
	}
	return c.protected
}

func (c *DBModuleConfig) LockDefaults() {
	c.DefaultsLocked = true
}

type AllProtectedModuleConfig struct {
	*DBModuleConfig
}

func (c AllProtectedModuleConfig) GetIsDefaultNotProtected(key string) (string, bool, error) {
	return "__ERROR", true, marvin.ErrConfProtected{Key: fmt.Sprintf("%s.%s", c.ModuleIdentifier, key)}
}
