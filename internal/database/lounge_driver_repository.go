package database

import (
	// "database/sql"
	// "fmt"
	// "time"

	// "github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	// "github.com/smarttransit/sms-auth-backend/internal/models"
)


// This handles the database operations for lounge driver
type loungeDriverRepository struct {
	db *sqlx.DB
}

// create a new lounge driver repository
func NewLoungeDriverRepository(db *sqlx.DB) *loungeDriverRepository{
	return &loungeDriverRepository{db:db}
}

