package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // postgresql
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/simplesurance/baur/storage"
)

// Client is a postgres storage client
type Client struct {
	db *sql.DB
}

// New establishes a connection a postgres db
func New(url string) (*Client, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Client{
		db: db,
	}, nil
}

// Close closes the connection
func (c *Client) Close() {
	c.db.Close()
}

// ListBuildsPerApp returns all builds for an app
func (c *Client) ListBuildsPerApp(appName string, maxResults int) ([]*storage.Build, error) {
	return nil, nil
}

func insertBuild(tx *sql.Tx, appID int, b *storage.Build) (int, error) {
	const stmt = `
	INSERT INTO build
	(application_id, start_timestamp, stop_timestamp, total_src_digest)
	VALUES($1, $2, $3, $4)
	RETURNING id;`

	var id int

	r := tx.QueryRow(stmt, appID, b.StartTimeStamp, b.StopTimeStamp, b.TotalInputDigest)

	if err := r.Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

func insertOutputIfNotExist(tx *sql.Tx, a *storage.Output) (int, error) {
	const insertStmt = `
	INSERT INTO output
	(name, type, digest, size_bytes)
	VALUES($1, $2, $3, $4)
	RETURNING id;
	`

	const selectStmt = `
	SELECT id FROM output
	WHERE name = $1 AND digest = $2 AND size_bytes = $3;
	`

	return insertIfNotExist(tx,
		insertStmt, []interface{}{a.Name, a.Type, a.Digest.String(), a.SizeBytes},
		selectStmt, []interface{}{a.Name, a.Digest.String(), a.SizeBytes})
}

func insertInputBuild(tx *sql.Tx, buildID, inputID int) error {
	const stmt = "INSERT into input_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, inputID)

	return err
}

func insertIfNotExist(
	tx *sql.Tx,
	insertStmt string,
	insertArgs []interface{},
	selectStmt string,
	selectArgs []interface{},
) (int, error) {
	var id int
	savepointName := xid.New().String()

	_, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepointName))
	if err != nil {
		return -1, errors.Wrapf(err, "creating savepoint %q failed", savepointName)
	}

	r := tx.QueryRow(insertStmt, insertArgs...)
	insertErr := r.Scan(&id)
	if insertErr == nil {
		return id, nil
	}

	// row already exist, TODO: only rollback and continue if it's
	// an already exist error
	_, err = tx.Exec(fmt.Sprintf("ROLLBACK TO %s", savepointName))
	if err != nil {
		return -1, errors.Wrapf(err, "rolling back transaction after insert error %q failed", insertErr)
	}

	r = tx.QueryRow(selectStmt, selectArgs...)
	if err := r.Scan(&id); err != nil {
		return -1, errors.Wrapf(err, "selecting input record failed after insert failed: %s", insertErr)
	}

	return id, nil
}

func insertInputIfNotExist(tx *sql.Tx, s *storage.Input) (int, error) {
	const insertStmt = `
	INSERT INTO input
	(url, digest)
	VALUES($1, $2)
	RETURNING id;
	`

	const selectStmt = `
	SELECT id FROM input
	WHERE url = $1 AND digest = $2;
	`

	return insertIfNotExist(tx,
		insertStmt, []interface{}{s.URL, s.Digest},
		selectStmt, []interface{}{s.URL, s.Digest})
}

func insertAppIfNotExist(tx *sql.Tx, appName string) (int, error) {
	const insertStmt = `
	INSERT INTO application
	(name)
	VALUES($1)
	RETURNING id;
	`
	const selectStmt = "SELECT id FROM application WHERE name = $1;"

	return insertIfNotExist(tx,
		insertStmt, []interface{}{appName},
		selectStmt, []interface{}{appName})
}

func insertOutputBuild(tx *sql.Tx, buildID, outputID int) error {
	const stmt = "INSERT into output_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, outputID)

	return err
}

func insertUpload(tx *sql.Tx, outputID int, url string, uploadDuration time.Duration) error {
	const stmt = `
	INSERT into upload
	(output_id, uri, upload_duration_msec)
	VALUES($1, $2, $3)
	RETURNING id
	`

	_, err := tx.Exec(stmt, outputID, url, uploadDuration/time.Millisecond)
	return err
}

func saveOutput(tx *sql.Tx, buildID int, a *storage.Output) error {
	outputID, err := insertOutputIfNotExist(tx, a)
	if err != nil {
		return errors.Wrap(err, "storing output record failed")
	}

	err = insertOutputBuild(tx, buildID, outputID)
	if err != nil {
		return errors.Wrap(err, "storing output_build record failed")
	}

	err = insertUpload(tx, outputID, a.URI, a.UploadDuration)
	if err != nil {
		return errors.Wrap(err, "storing upload record failed")
	}

	return nil
}

// Save stores a build
func (c *Client) Save(b *storage.Build) error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.Wrap(err, "starting transaction failed")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	appID, err := insertAppIfNotExist(tx, b.AppNameLower())
	if err != nil {
		return errors.Wrap(err, "storing application record failed")
	}

	buildID, err := insertBuild(tx, appID, b)
	if err != nil {
		return errors.Wrap(err, "storing build record failed")
	}

	for _, a := range b.Outputs {
		if err := saveOutput(tx, buildID, a); err != nil {
			return err
		}
	}

	for _, s := range b.Inputs {
		inputID, err := insertInputIfNotExist(tx, s)
		if err != nil {
			return errors.Wrap(err, "storing input record failed")
		}

		err = insertInputBuild(tx, buildID, inputID)
		if err != nil {
			return errors.Wrap(err, "storing input_build failed")
		}
	}

	return nil
}
