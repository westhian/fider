package dbx

import (
	"context"
	"database/sql"
	"io/ioutil"
	"reflect"
	"time"

	"strings"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/log"

	//required
	_ "github.com/lib/pq"
)

// New creates a new Database instance without logging
func New() *Database {
	conn, err := sql.Open("postgres", env.Config.Database.URL)
	if err != nil {
		panic(errors.Wrap(err, "failed to open connection to the database"))
	}

	conn.SetMaxIdleConns(env.Config.Database.MaxIdleConns)
	conn.SetMaxOpenConns(env.Config.Database.MaxOpenConns)

	return &Database{conn, NewRowMapper()}
}

// Database represents a connection to a SQL database
type Database struct {
	conn   *sql.DB
	mapper *RowMapper
}

// Connection returns current database connection
func (db *Database) Connection() *sql.DB {
	return db.conn
}

// Ping checks if current database connection is healthy
func (db *Database) Ping() error {
	_, err := db.conn.Exec("SELECT 1")
	return err
}

// Begin returns a new SQL transaction
func (db *Database) Begin(ctx context.Context) (*Trx, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start new transaction")
	}
	return &Trx{tx: tx, ctx: ctx, mapper: db.mapper}, nil
}

// Close connection to database
func (db *Database) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *Database) load(path string) {
	content, err := ioutil.ReadFile(env.Path(path))
	if err != nil {
		panic(errors.Wrap(err, "failed to read file %s", path))
	}

	_, err = db.conn.Exec(string(content))
	if err != nil {
		panic(errors.Wrap(err, "failed to execute %s", path))
	}
}

// Seed clean and insert new seed data for testing
func (db *Database) Seed() {
	if env.IsTest() {
		db.load("/app/pkg/dbx/setup.sql")
	}
}

//Trx represents a Database transaction
type Trx struct {
	tx     *sql.Tx
	ctx    context.Context
	mapper *RowMapper
}

var formatter = strings.NewReplacer("\t", "", "\n", " ")

// Execute given SQL command
func (trx *Trx) Execute(command string, args ...interface{}) (int64, error) {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	result, err := trx.tx.ExecContext(trx.ctx, command, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute trx.Execute")
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// Scalar returns first row and first column
func (trx *Trx) Scalar(data interface{}, command string, args ...interface{}) error {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	row := trx.tx.QueryRowContext(trx.ctx, command, args...)
	err := row.Scan(data)
	if err != nil {
		if err == sql.ErrNoRows {
			return app.ErrNotFound
		}
		return errors.Wrap(err, "failed to execute trx.Scalar")
	}
	return nil
}

// Get first row and bind to given data
func (trx *Trx) Get(data interface{}, command string, args ...interface{}) error {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	rows, err := trx.tx.QueryContext(trx.ctx, command, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute trx.Get")
	}

	defer rows.Close()
	if rows.Next() {
		columns, _ := rows.Columns()
		err := trx.mapper.Map(data, columns, rows.Scan)
		if err != nil {
			return errors.Wrap(err, "failed to map result to model")
		}
		return nil
	}

	return app.ErrNotFound
}

// Exists returns true if at least one record is found
func (trx *Trx) Exists(command string, args ...interface{}) (bool, error) {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	rows, err := trx.tx.QueryContext(trx.ctx, command, args...)
	if err != nil {
		return false, errors.Wrap(err, "failed to execute trx.Exists")
	}

	defer rows.Close()
	return rows.Next(), nil
}

// Count returns number of rows
func (trx *Trx) Count(command string, args ...interface{}) (int, error) {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	rows, err := trx.tx.QueryContext(trx.ctx, command, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute trx.Count")
	}

	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	return count, nil
}

//Select all matched rows bind to given data
func (trx *Trx) Select(data interface{}, command string, args ...interface{}) error {
	if log.IsEnabled(log.DEBUG) {
		command = formatter.Replace(command)
		start := time.Now()
		defer func() {
			log.Debugf(trx.ctx, "@{Command:yellow} @{Args:blue} executed in @{ElapsedMs:magenta}ms", dto.Props{
				"Command":   command,
				"Args":      args,
				"ElapsedMs": time.Since(start).Nanoseconds() / int64(time.Millisecond),
			})
		}()
	}

	rows, err := trx.tx.QueryContext(trx.ctx, command, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute trx.Select")
	}

	defer rows.Close()
	sliceType := reflect.TypeOf(data).Elem()
	items := reflect.New(sliceType).Elem()
	itemType := sliceType.Elem().Elem()
	var columns []string
	for rows.Next() {
		if columns == nil {
			columns, _ = rows.Columns()
		}
		item := reflect.New(itemType)
		if err = trx.mapper.Map(item.Interface(), columns, rows.Scan); err != nil {
			return errors.Wrap(err, "failed to map result to model")
		}
		items = reflect.Append(items, item)
	}

	if items.Len() > 0 {
		reflect.Indirect(reflect.ValueOf(data)).Set(items)
	}
	return nil
}

// Commit current transaction
func (trx *Trx) Commit() error {
	err := trx.tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

// Rollback current transaction
func (trx *Trx) Rollback() error {
	err := trx.tx.Rollback()
	if err != nil {
		return errors.Wrap(err, "failed to rollback transaction")
	}
	return nil
}

func IsCanceledStmpByUser(err error) bool {
	if err != nil {
		return err.Error() == "pq: canceling statement due to user request"
	}
	return false
}
