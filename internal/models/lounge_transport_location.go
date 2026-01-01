package models

import (
	"time"
	"github.com/google/uuid"
)

type LoungeTransportLocation struct {

	// Basic details of the transport location
	ID uuid.UUID `db:"id" json:"id"`
	LoungeID uuid.UUID `db:"lounge_id" json:"lounge_id"`
	Location string `db:"location" json:"location"`

	// LocationStatus defines the availability of a transport location
	Status LocationStatus `db:"status" json:"status"`

	// CreatedAt and UpdatedAt for the ease of maintenance
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

}

// creating location status ENUM
type LocationStatus string 

const (
	LocationStatusActive LocationStatus = "active"
	LocationStatusInactive LocationStatus = "inactive"
)
