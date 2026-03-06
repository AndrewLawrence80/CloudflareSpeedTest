package store

import (
	"sync"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/internal/model"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// getDB is the process-wide singleton accessor for the SQLite database.
// It panics on initialisation failure because a missing database is
// unrecoverable and far better caught at startup than as a nil-pointer
// dereference deep inside the application.
var getDB = sync.OnceValue(func() *gorm.DB {
	db, err := newDB(common.EnvOr("DB_PATH", "./speedtest.db"))
	if err != nil {
		log.GetLogger().Error("failed to open database", "error", err)
		panic(err)
	}
	return db
})

func GetDB() *gorm.DB {
	return getDB()
}

// newDB opens (or creates) a SQLite database at the given file path,
// runs auto-migration and returns a ready-to-use *gorm.DB.
func newDB(path string) (retDB *gorm.DB, retErr error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: newSlogGormLogger(log.GetLogger()),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Close the underlying connection if any subsequent setup step fails,
	// preventing a connection leak.
	defer func() {
		if retErr != nil {
			sqlDB.Close()
		}
	}()

	// SQLite does not support concurrent writes. A single connection prevents
	// "database is locked" errors within this process.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// WAL mode improves read concurrency from external processes and tools
	// even when MaxOpenConns=1 is used within this process.
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, err
	}
	// Wait up to 10 s before returning SQLITE_BUSY, as belt-and-suspenders
	// alongside MaxOpenConns=1.
	if err := db.Exec("PRAGMA busy_timeout=10000;").Error; err != nil {
		return nil, err
	}
	// SQLite disables FK enforcement by default; enable it explicitly.
	if err := db.Exec("PRAGMA foreign_keys=ON;").Error; err != nil {
		return nil, err
	}

	if err := model.AutoMigrate(db); err != nil {
		return nil, err
	}
	return db, nil
}
