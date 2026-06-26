//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgresql://postgres.pttatcukzpceljcrwehk:KQ95tJUYdFX251VR@aws-1-us-east-1.pooler.supabase.com:5432/postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check lounge owner details for user '4d54f70d-b75c-46fd-9f7a-82423e8a0c60'
	userID := "4d54f70d-b75c-46fd-9f7a-82423e8a0c60"
	fmt.Printf("--- Checking Lounge Owner status for User %s ---\n", userID)
	var ownerID, status string
	err = db.QueryRow(`
		SELECT id, verification_status 
		FROM lounge_owners 
		WHERE user_id = $1
	`, userID).Scan(&ownerID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No lounge owner record found for this user ID.")
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Lounge Owner ID: %s, Verification Status: %s\n", ownerID, status)
	}

	// Check owner of lounge 'e528941b-3c29-4b4e-87e8-6c2ae4ea8c34'
	loungeID := "e528941b-3c29-4b4e-87e8-6c2ae4ea8c34"
	fmt.Printf("\n--- Checking Lounge details for Lounge %s ---\n", loungeID)
	var lOwnerID string
	err = db.QueryRow(`
		SELECT lounge_owner_id 
		FROM lounges 
		WHERE id = $1
	`, loungeID).Scan(&lOwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Lounge not found.")
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Lounge Owner ID associated with Lounge: %s\n", lOwnerID)
		
		// Check that owner's verification status
		var verificationStatus string
		err = db.QueryRow(`
			SELECT verification_status 
			FROM lounge_owners 
			WHERE id = $1
		`, lOwnerID).Scan(&verificationStatus)
		if err != nil {
			fmt.Printf("Error getting verification status of owner: %v\n", err)
		} else {
			fmt.Printf("Owner's Verification Status: %s\n", verificationStatus)
		}
	}
}
