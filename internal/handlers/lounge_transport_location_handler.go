package handlers

import (

	// "log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smarttransit/sms-auth-backend/internal/database"
	"github.com/smarttransit/sms-auth-backend/internal/middleware"


)

// LoungeTransportLocationHandler handles HTTP requests related to lounge transport locations
type LoungeTransportLocationHandler struct{

	loungeOwnerRepo *database.LoungeOwnerRepository
	loungeRepo *database.LoungeRepository
	loungeTransportLocationRepo *database.LoungeTransportLocationRepository

}

// NewLoungeTransportLocationHandler creates new lounge transport location handler
func NewLoungeTransportLocationHandler(

	loungeOwnerRepo *database.LoungeOwnerRepository,
	loungeRepo *database.LoungeRepository,
	loungeTransportLocationRepo *database.LoungeTransportLocationRepository,

) *LoungeTransportLocationHandler{
	return &LoungeTransportLocationHandler{
		loungeOwnerRepo: loungeOwnerRepo,
		loungeRepo: loungeRepo,
		loungeTransportLocationRepo: loungeTransportLocationRepo,
	}
}

type AddLoungeTransportLocationRequest struct{
	Location string `json:"location" binding:"required"`
}


// Add transpport locations to lounge
func (h *LoungeTransportLocationHandler) AddLoungeTransportLocation(c *gin.Context){

	// creating a struct variable to store request
	var req AddLoungeTransportLocationRequest

	// getting the data fed by the user
	if err := c.ShouldBindJSON(&req); err!=nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Get user context from JWT middleware
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	// getting the loungeID
	loungeID := uuid.MustParse(c.Param("id"))

	// Get lounge owner record
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Lounge owner not found",
		})
		return
	}

	// Verify lounge ownership
	lounge, err := h.loungeRepo.GetLoungeByID(loungeID)
	if err != nil || lounge == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Lounge not found",
		})
		return
	}

	if lounge.LoungeOwnerID != owner.ID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to add transport locations for this lounge",
		})
		return
	}

	// // Add transport location to lounge
	// AddedTransportLocation,err:= h.loungeTransportLocationRepo.AddTransportLocationToLounge()

}
