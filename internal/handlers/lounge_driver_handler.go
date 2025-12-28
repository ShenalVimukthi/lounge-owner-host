package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smarttransit/sms-auth-backend/internal/database"
	"github.com/smarttransit/sms-auth-backend/internal/middleware"
	"github.com/smarttransit/sms-auth-backend/internal/models"
)

// lounge driver handles the HTTP requests related to lounge driver
type LoungeDriverHandler struct {
	loungeOwnerRepo  *database.LoungeOwnerRepository
	loungeRepo       *database.LoungeRepository
	loungeDriverRepo *database.LoungeDriverRepository
}

// createing a new lounge staff handler instance
func NewLoungeDriverHandler(
	loungeOwnerRepo *database.LoungeOwnerRepository,
	loungeRepo *database.LoungeRepository,
	loungeDriverRepo *database.LoungeDriverRepository,
) *LoungeDriverHandler {
	return &LoungeDriverHandler{
		loungeOwnerRepo:  loungeOwnerRepo,
		loungeRepo:       loungeRepo,
		loungeDriverRepo: loungeDriverRepo,
	}
}

// add drivers to lounge
type AddDriverRequest struct {
	// getting user data to the struct by using binding for safety
	Name          string                   `json:"name" binding:"required,min=2,max=100"`
	NIC           string                   `json:"nic_number" binding:"required"`
	ContactNumber string                   `json:"contact_no" binding:"required"`
	VehicleNumber string                   `json:"vehicle_no" binding:"required"`
	VehicleType   models.DriverVehicleType `json:"vehicle_type" binding:"required,oneof=three_wheeler car van"`
}

// adding drivers to the lounge

func (h *LoungeDriverHandler) AddDriver(c *gin.Context) {
	// creating the req struct instnace inorder to get the data inside the struct
	var req AddDriverRequest

	// binding the request to the struct if error comes returning the error
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// validating the user who is sending the request
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	// extracting the loungeID from the URL (also handling any errors)
	loungeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid lounge_id",
			Message: "Invalid lounge ID format",
		})
		return
	}

	// get lounge owner record
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil {
		log.Printf("ERROR: Failed to get owner for user %s: %v", userCtx.UserID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to verify ownership",
		})
	}
	if owner == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Lounge owner not found",
		})
		return
	}

	// verify lounge ownership
	lounge, err := h.loungeRepo.GetLoungeByID(loungeID)
	if err != nil {
		log.Printf("ERROR: Failed to get lounge %s: %v", loungeID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to verify lounge",
		})
		return
	}
	if lounge == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Lounge not found",
		})
		return
	}
	if lounge.LoungeOwnerID != owner.ID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to add drivers to this lounge",
		})
		return
	}

	// converting the data into the model to feed into repository function
	driver := &models.LoungeDriver{
		LoungeID:      loungeID,
		Name:          req.Name,
		NIC:           req.NIC,
		ContactNumber: req.ContactNumber,
		VehicleNumber: req.VehicleNumber,
		VehicleType:   req.VehicleType,
		Status:        models.DriverStatusActive, //default status is active for driver
	}

	// saving using the repository
	savedDriver, err := h.loungeDriverRepo.AddDriver(driver)

	// handling the errors gracefully
	if err != nil {
		log.Printf("ERROR: Failed to add driver for lounge %s: %v", loungeID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "add_failed",
			Message: "Failed to add driver",
		})
		return
	}

	// if all good then sending statusCreated with the data struct for reference
	c.JSON(http.StatusCreated, savedDriver)
}
