package pkg

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type CheckSummer struct {
	tempDir   string
	srcDB     *sql.DB
	srcDBPath string
}

var (
	messagesDB           = "msgindex.db"
	msgIndexMigrateStmt  = "INSERT INTO messages SELECT * FROM src.messages WHERE epoch >= ? AND epoch =< ?"
	sha3sumDotCommand    = ".sha3sum"
	findMsgIndexGapsStmt = `SELECT epoch + 1 AS first_missing, (next_nc - 1) AS last_missing
FROM (SELECT epoch, LEAD(epoch) OVER (ORDER BY epoch) AS next_nc FROM src.messages WHERE epoch >= ? AND epoch <= ?) h
WHERE next_nc > epoch + 1`
	checkRangeIsPopulatedStmt = fmt.Sprintf("SELECT EXISTS(%s)", findGapsStmt)
)

// from lotus chain/store/sqlite/msgindex.go
var msgIndexDBDefs = []string{
	`CREATE TABLE IF NOT EXISTS messages (
     cid VARCHAR(80) PRIMARY KEY ON CONFLICT REPLACE,
     tipset_cid VARCHAR(80) NOT NULL,
     epoch INTEGER NOT NULL
   )`,
	`CREATE INDEX IF NOT EXISTS tipset_cids ON messages (tipset_cid)`,
	`CREATE INDEX IF NOT EXISTS tipset_epochs ON messages (epoch)`,
	`CREATE TABLE IF NOT EXISTS _meta (
    	version UINT64 NOT NULL UNIQUE
	)`,
	`INSERT OR IGNORE INTO _meta (version) VALUES (1)`,
}

func NewChecksummer(srcDir string) (*CheckSummer, error) {
	// Create a temporary directory to hold the SQLite database file
	tempDir, err := os.MkdirTemp("", "temp_db")
	if err != nil {
		return nil, err
	}

	srcDBPath := filepath.Join(srcDir, messagesDB)
	srcDB, err := sql.Open("sqlite3", srcDBPath+"?mode=rwc")
	if err != nil {
		return nil, err
	}
	return &CheckSummer{
		srcDBPath: filepath.Join(srcDir, messagesDB),
		srcDB:     srcDB,
		tempDir:   tempDir,
	}, nil
}

func (m *CheckSummer) FindGaps(start, stop uint) ([][2]uint, error) {
	rows, err := m.srcDB.Query(findMsgIndexGapsStmt, start, stop)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.Infof("No gaps found for range %d to %d", start, stop)
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var gaps [][2]uint
	for rows.Next() {
		var gapStart, gapStop uint
		if err := rows.Scan(&gapStart, &gapStop); err != nil {
			return nil, err
		}
		gaps = append(gaps, [2]uint{gapStart, gapStop})
	}
	return gaps, nil
}

// Checksum checksums a chunk defined by the start and stop epochs (inclusive)
func (m *CheckSummer) Checksum(start, stop uint) (string, error) {
	// Create the path for the temporary database file
	tempDBPath := filepath.Join(m.tempDir, messagesDB)
	dstMsgDB, err := sql.Open("sqlite3", tempDBPath+"?mode=rwc")
	if err != nil {
		return "", xerrors.Errorf("open sqlite3 database: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDBPath); err != nil {
			logrus.Errorf("remove temp db: %s", err)
		}
	}()
	defer func() {
		if err := dstMsgDB.Close(); err != nil {
			logrus.Errorf("close temp db: %s", err)
		}
	}()

	// Create the temporary dst messages table
	for _, stmt := range msgIndexDBDefs {
		_, err = dstMsgDB.Exec(stmt)
		if err != nil {
			return "", xerrors.Errorf("create temp msgindex schema (stmt: %s): %w", stmt, err)
		}
	}

	_, err = dstMsgDB.Exec("ATTACH DATABASE ? AS src", m.srcDBPath)
	if err != nil {
		return "", xerrors.Errorf("attach src database: %w", err)
	}
	_, err = dstMsgDB.Exec(msgIndexMigrateStmt, start, stop)
	if err != nil {
		return "", xerrors.Errorf("migrate into dst.messages: %w", err)
	}
	_, err = dstMsgDB.Exec("DETACH src")
	if err != nil {
		return "", xerrors.Errorf("detach src database: %w", err)
	}
	var hash string
	return hash, dstMsgDB.QueryRow(sha3sumDotCommand).Scan(&hash)
}

// Close closes the temporary database file and removes the temporary directory
func (m *CheckSummer) Close() error {
	return os.RemoveAll(m.tempDir)
}
