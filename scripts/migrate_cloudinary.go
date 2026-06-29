package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Old Cloud Name to identify which URLs need migrating
const oldCloudName = "dvg5cu5uv"

func main() {
	dryRun := flag.Bool("dry-run", false, "Preview migration without performing uploads or DB updates")
	flag.Parse()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		log.Fatal("CLOUDINARY_URL environment variable is required")
	}

	// Parse new Cloudinary URL
	parsedURL, err := url.Parse(cloudinaryURL)
	if err != nil || parsedURL.Scheme != "cloudinary" || parsedURL.User == nil {
		log.Fatalf("Invalid CLOUDINARY_URL format: %v", err)
	}

	newAPIKey := parsedURL.User.Username()
	newAPISecret, _ := parsedURL.User.Password()
	newCloudName := parsedURL.Host

	if newAPIKey == "" || newAPISecret == "" || newCloudName == "" {
		log.Fatal("CLOUDINARY_URL must include api key, api secret, and cloud name")
	}

	if newCloudName == oldCloudName {
		log.Fatalf("Error: New cloud name '%s' is the same as the old cloud name. Please update CLOUDINARY_URL in .env first.", oldCloudName)
	}

	log.Printf("Starting Cloudinary migration from '%s' to '%s'...", oldCloudName, newCloudName)
	if *dryRun {
		log.Println("⚠️  DRY RUN MODE ENABLED: No changes will be saved to the database or uploaded to Cloudinary.")
	}

	// Connect to Database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Connected to PostgreSQL successfully.")

	// Counters
	var totalFound, totalMigrated, totalErrors int

	migrateSingleColumn := func(tableName, idCol, valCol string) {
		query := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s LIKE '%%/%s/%%'", idCol, valCol, tableName, valCol, oldCloudName)
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error querying %s.%s: %v", tableName, valCol, err)
			return
		}
		defer rows.Close()

		type record struct {
			id  string
			val string
		}
		var records []record
		for rows.Next() {
			var id, val sql.NullString
			if err := rows.Scan(&id, &val); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			if id.Valid && val.Valid {
				records = append(records, record{id: id.String, val: val.String})
			}
		}

		if len(records) == 0 {
			return
		}

		log.Printf("Found %d records in '%s.%s' requiring migration.", len(records), tableName, valCol)

		for _, rec := range records {
			totalFound++
			log.Printf("[%s] Migrating %s ID %s: %s", tableName, idCol, rec.id, rec.val)

			if *dryRun {
				continue
			}

			newURL, err := uploadFromURL(rec.val, newCloudName, newAPIKey, newAPISecret)
			if err != nil {
				log.Printf("❌ Failed to migrate URL '%s': %v", rec.val, err)
				totalErrors++
				continue
			}

			updateQuery := fmt.Sprintf("UPDATE %s SET %s = $1, updated_at = NOW() WHERE %s = $2", tableName, valCol, idCol)
			// Handle tables without standard updated_at column
			if tableName == "lounge_booking_pre_orders" {
				updateQuery = fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", tableName, valCol, idCol)
			}
			_, err = db.Exec(updateQuery, newURL, rec.id)
			if err != nil {
				log.Printf("❌ Failed to update DB for %s ID %s: %v", tableName, rec.id, err)
				totalErrors++
				continue
			}

			log.Printf("✅ Successfully migrated to: %s", newURL)
			totalMigrated++
		}
	}

	// 1. Migrate lounge_products.image_url
	migrateSingleColumn("lounge_products", "id", "image_url")

	// 2. Migrate lounge_products.thumbnail_url
	// 3. Migrate lounge_special_packages.image_url
	migrateSingleColumn("lounge_special_packages", "id", "image_url")

	// 4. Migrate users.profile_photo_url
	migrateSingleColumn("users", "id", "profile_photo_url")

	// 7. Migrate passengers.profile_photo_url
	migrateSingleColumn("passengers", "id", "profile_photo_url")

	// 8. Migrate lounge_booking_pre_orders.product_image_url
	migrateSingleColumn("lounge_booking_pre_orders", "id", "product_image_url")

	// 9. Migrate lounges.images (JSONB Array)
	{
		tableName := "lounges"
		idCol := "id"
		valCol := "images"
		query := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s::text LIKE '%%/%s/%%'", idCol, valCol, tableName, valCol, oldCloudName)
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error querying %s.%s: %v", tableName, valCol, err)
		} else {
			defer rows.Close()

			type loungeRecord struct {
				id     string
				images []byte
			}
			var records []loungeRecord
			for rows.Next() {
				var id string
				var images []byte
				if err := rows.Scan(&id, &images); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}
				records = append(records, loungeRecord{id: id, images: images})
			}

			if len(records) > 0 {
				log.Printf("Found %d records in '%s.%s' requiring migration.", len(records), tableName, valCol)

				for _, rec := range records {
					var urls []string
					if err := json.Unmarshal(rec.images, &urls); err != nil {
						log.Printf("Error parsing JSON images array for lounge %s: %v", rec.id, err)
						continue
					}

					updated := false
					for i, oldURL := range urls {
						if strings.Contains(oldURL, "/"+oldCloudName+"/") {
							totalFound++
							log.Printf("[%s] Migrating image array item %d for Lounge ID %s: %s", tableName, i, rec.id, oldURL)

							if *dryRun {
								continue
							}

							newURL, err := uploadFromURL(oldURL, newCloudName, newAPIKey, newAPISecret)
							if err != nil {
								log.Printf("❌ Failed to migrate URL '%s': %v", oldURL, err)
								totalErrors++
								continue
							}

							urls[i] = newURL
							updated = true
							totalMigrated++
						}
					}

					if updated && !*dryRun {
						newJSON, err := json.Marshal(urls)
						if err != nil {
							log.Printf("❌ Failed to serialize updated images array for Lounge %s: %v", rec.id, err)
							continue
						}

						updateQuery := fmt.Sprintf("UPDATE %s SET %s = $1, updated_at = NOW() WHERE %s = $2", tableName, valCol, idCol)
						_, err = db.Exec(updateQuery, newJSON, rec.id)
						if err != nil {
							log.Printf("❌ Failed to update DB for Lounge %s: %v", rec.id, err)
							continue
						}
						log.Printf("✅ Successfully updated Lounge %s image array.", rec.id)
					}
				}
			}
		}
	}

	log.Printf("Cloudinary Migration Completed.")
	log.Printf("Summary: Found %d urls. Migrated %d urls successfully. Errors encountered: %d.", totalFound, totalMigrated, totalErrors)
}

// uploadFromURL signs parameters and uploads a remote image URL to the new Cloudinary account
func uploadFromURL(oldURL, newCloudName, newAPIKey, newAPISecret string) (string, error) {
	publicID, err := extractPublicID(oldURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract public ID: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	params := map[string]string{
		"overwrite": "true",
		"public_id": publicID,
		"timestamp": timestamp,
	}

	signature := sign(params, newAPISecret)

	form := url.Values{}
	form.Set("file", oldURL)
	form.Set("public_id", publicID)
	form.Set("overwrite", "true")
	form.Set("timestamp", timestamp)
	form.Set("api_key", newAPIKey)
	form.Set("signature", signature)

	apiURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", newCloudName)
	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return "", fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloudinary upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SecureURL string `json:"secure_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return result.SecureURL, nil
}

func extractPublicID(imageURL string) (string, error) {
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}

	segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	uploadIndex := -1
	for index, segment := range segments {
		if segment == "upload" {
			uploadIndex = index
			break
		}
	}
	if uploadIndex == -1 || uploadIndex+1 >= len(segments) {
		return "", fmt.Errorf("unsupported URL format: %s", imageURL)
	}

	assetSegments := segments[uploadIndex+1:]
	for len(assetSegments) > 0 && isVersionSegment(assetSegments[0]) {
		assetSegments = assetSegments[1:]
	}
	if len(assetSegments) == 0 {
		return "", fmt.Errorf("unsupported URL format: %s", imageURL)
	}

	assetPath := strings.Join(assetSegments, "/")
	extension := path.Ext(assetPath)
	return strings.TrimSuffix(assetPath, extension), nil
}

func isVersionSegment(segment string) bool {
	if len(segment) < 2 || segment[0] != 'v' {
		return false
	}
	_, err := strconv.ParseInt(segment[1:], 10, 64)
	return err == nil
}

func sign(params map[string]string, apiSecret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(params[key])
	}
	builder.WriteString(apiSecret)
	checksum := sha1.Sum([]byte(builder.String()))
	return hex.EncodeToString(checksum[:])
}
