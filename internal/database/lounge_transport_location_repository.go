package database

import (
	// "database/sql"
	"log"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	// "github.com/pelletier/go-toml/query"
	"github.com/smarttransit/sms-auth-backend/internal/models"
)

// Lounge Transport Location Repository handles database operations of lounge transport locations
type LoungeTransportLocationRepository struct{
	db *sqlx.DB
}

// NewLoungeTransportLocationRepository creates new lounge transport location repository
func NewLoungeTransportLocationRepository(db *sqlx.DB) *LoungeTransportLocationRepository{
	return & LoungeTransportLocationRepository{db: db}
}

// Add Transport Locations for lounge 
func (r *LoungeTransportLocationRepository) AddTransportLocationToLounge(TransportLocation models.LoungeTransportLocation)(*models.LoungeTransportLocation,error){
	
	// setting auto-generated and default fields
	TransportLocation.ID = uuid.New()
	TransportLocation.CreatedAt = time.Now()
	TransportLocation.UpdatedAt = time.Now()

	// updating the status for the location default is active 
	if TransportLocation.Status == ""{
		TransportLocation.Status = models.LoungeTransportLocationStatusActive
	}

	// setting up the query to run
	query := `INSERT INTO lounge_transport_locations (
			 id, lounge_id, location, status, created_at, updated_at)
			 VALUES($1, $2, $3, $4, $5, $6)
			 RETURNING id, created_at, updated_at`
	
    err := r.db.QueryRowx(query,
		TransportLocation.ID,
		TransportLocation.LoungeID,
		TransportLocation.Location,
		TransportLocation.Status,
		TransportLocation.CreatedAt,
		TransportLocation.UpdatedAt,
	).Scan(&TransportLocation.ID,&TransportLocation.CreatedAt,&TransportLocation.UpdatedAt)

	if err != nil {
		log.Printf("ERROR: Failed to add location for lounge %s: %v", TransportLocation.LoungeID, err)
		return nil, fmt.Errorf("failed to add location: %w", err)
	}

	// if okay sending the created transport location data
	return &TransportLocation, nil

}