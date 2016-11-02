package database

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type Conn struct {
	*sql.DB
}

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

func (c *Conn) SyntaxCheck(query ...string) {
	for _, v := range query {
		stmt, err := c.DB.Prepare(v)
		if err != nil {
			panic(errors.Wrap(err, "SQL syntax check failed"))
		}
		stmt.Close()
	}
}
