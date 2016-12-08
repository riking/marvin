package database

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type Conn struct {
	*sql.DB
}

// Dial constructs a database connection for Marvin.
func Dial(connect string) (*Conn, error) {
	db, err := sql.Open("postgres", connect)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect")
	}
	c := &Conn{
		DB: db,
	}
	err = c.setupMigrate()
	if err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to setup database")
	}
	return c, nil
}

// SyntaxCheck will attempt to prepare every statement passed as a parameter,
// and panic if any of them cause a syntax error.
//
// This should be called at module Load() time.
func (c *Conn) SyntaxCheck(query ...string) {
	for _, v := range query {
		stmt, err := c.DB.Prepare(v)
		if err != nil {
			panic(errors.Wrap(err, "SQL syntax check failed"))
		}
		stmt.Close()
	}
}
