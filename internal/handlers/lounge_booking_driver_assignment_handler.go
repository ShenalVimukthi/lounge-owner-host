package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smarttransit/sms-auth-backend/internal/database"
	"github.com/smarttransit/sms-auth-backend/internal/middleware"
	"github.com/smarttransit/sms-auth-backend/internal/models"
)

// LoungeBookingDriverAssignmentHandler handles driver assignment operations
type LoungeBookingDriverAssignmentHandler struct {
	assignmentRepo  *database.LoungeBookingDriverAssignmentRepository
	loungeOwnerRepo *database.LoungeOwnerRepository
	loungeRepo      *database.LoungeRepository
}

// NewLoungeBookingDriverAssignmentHandler creates a new handler
func NewLoungeBookingDriverAssignmentHandler(
	assignmentRepo *database.LoungeBookingDriverAssignmentRepository,
	loungeOwnerRepo *database.LoungeOwnerRepository,
	loungeRepo *database.LoungeRepository,
) *LoungeBookingDriverAssignmentHandler {
	return &LoungeBookingDriverAssignmentHandler{
		assignmentRepo:  assignmentRepo,
		loungeOwnerRepo: loungeOwnerRepo,
		loungeRepo:      loungeRepo,
	}
}

// CreateAssignment handles POST /api/v1/lounge-booking-driver-assignments
func (h *LoungeBookingDriverAssignmentHandler) CreateAssignment(c *gin.Context) {
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	var req models.CreateLoungeBookingDriverAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Verify lounge ownership
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Not a lounge owner",
		})
		return
	}

	lounge, err := h.loungeRepo.GetLoungeByID(req.LoungeID)
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
			Message: "You don't own this lounge",
		})
		return
	}

	assignment := &models.LoungeBookingDriverAssignment{
		ID:              uuid.New(),
		LoungeID:        req.LoungeID,
		DriverID:        req.DriverID,
		LoungeBookingID: req.LoungeBookingID,
		GuestName:       req.GuestName,
		GuestContact:    req.GuestContact,
		DriverContact:   req.DriverContact,
		Status:          models.DriverAssignmentStatusPending,
	}

	if err := h.assignmentRepo.CreateAssignment(assignment); err != nil {
		log.Printf("ERROR: Failed to create driver assignment: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create assignment",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Driver assignment created successfully",
		"assignment":  assignment,
	})
}

// GetAssignmentByID handles GET /api/v1/lounge-booking-driver-assignments/:id
func (h *LoungeBookingDriverAssignmentHandler) GetAssignmentByID(c *gin.Context) {
	assignmentIDStr := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	assignment, err := h.assignmentRepo.GetAssignmentByID(assignmentID)
	if err != nil || assignment == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Assignment not found",
		})
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// GetAssignmentsByBooking handles GET /api/v1/lounge-bookings/:id/driver-assignments
func (h *LoungeBookingDriverAssignmentHandler) GetAssignmentsByBooking(c *gin.Context) {
	bookingIDStr := c.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid booking ID format",
		})
		return
	}

	assignments, err := h.assignmentRepo.GetAssignmentByBookingID(bookingID)
	if err != nil {
		log.Printf("ERROR: Failed to get assignments for booking: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve assignments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"total":       len(assignments),
	})
}

// GetAssignmentsByDriver handles GET /api/v1/drivers/:driver_id/assignments
func (h *LoungeBookingDriverAssignmentHandler) GetAssignmentsByDriver(c *gin.Context) {
	driverIDStr := c.Param("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid driver ID format",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	statusQuery := c.Query("status")
	var status *string
	if statusQuery != "" {
		status = &statusQuery
	}

	assignments, err := h.assignmentRepo.GetAssignmentsByDriverID(driverID, status, limit, offset)
	if err != nil {
		log.Printf("ERROR: Failed to get driver assignments: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve assignments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetAssignmentsByLounge handles GET /api/v1/lounges/:id/driver-assignments
func (h *LoungeBookingDriverAssignmentHandler) GetAssignmentsByLounge(c *gin.Context) {
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	loungeIDStr := c.Param("id")
	loungeID, err := uuid.Parse(loungeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lounge ID format",
		})
		return
	}

	// Verify lounge ownership
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Not a lounge owner",
		})
		return
	}

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
			Message: "You don't own this lounge",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	statusQuery := c.Query("status")
	var status *string
	if statusQuery != "" {
		status = &statusQuery
	}

	assignments, err := h.assignmentRepo.GetAssignmentsByLoungeID(loungeID, status, limit, offset)
	if err != nil {
		log.Printf("ERROR: Failed to get lounge driver assignments: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve assignments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"lounge_id":   loungeID,
		"limit":       limit,
		"offset":      offset,
	})
}

// UpdateAssignment handles PUT /api/v1/lounge-booking-driver-assignments/:id
func (h *LoungeBookingDriverAssignmentHandler) UpdateAssignment(c *gin.Context) {
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	assignmentIDStr := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	// Get the assignment
	assignment, err := h.assignmentRepo.GetAssignmentByID(assignmentID)
	if err != nil || assignment == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Assignment not found",
		})
		return
	}

	// Verify lounge ownership
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Not a lounge owner",
		})
		return
	}

	lounge, err := h.loungeRepo.GetLoungeByID(assignment.LoungeID)
	if err != nil || lounge == nil || lounge.LoungeOwnerID != owner.ID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You don't own this lounge",
		})
		return
	}

	var req models.UpdateLoungeBookingDriverAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.assignmentRepo.UpdateAssignment(assignmentID, &req); err != nil {
		log.Printf("ERROR: Failed to update assignment: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to update assignment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Assignment updated successfully",
		"assignment_id": assignmentID,
	})
}

// DeleteAssignment handles DELETE /api/v1/lounge-booking-driver-assignments/:id
func (h *LoungeBookingDriverAssignmentHandler) DeleteAssignment(c *gin.Context) {
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	assignmentIDStr := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	// Get the assignment
	assignment, err := h.assignmentRepo.GetAssignmentByID(assignmentID)
	if err != nil || assignment == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Assignment not found",
		})
		return
	}

	// Verify lounge ownership
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Not a lounge owner",
		})
		return
	}

	lounge, err := h.loungeRepo.GetLoungeByID(assignment.LoungeID)
	if err != nil || lounge == nil || lounge.LoungeOwnerID != owner.ID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You don't own this lounge",
		})
		return
	}

	if err := h.assignmentRepo.DeleteAssignment(assignmentID); err != nil {
		log.Printf("ERROR: Failed to delete assignment: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to delete assignment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Assignment deleted successfully",
		"assignment_id": assignmentID,
	})
}

// CancelAssignment handles POST /api/v1/lounge-booking-driver-assignments/:id/cancel
func (h *LoungeBookingDriverAssignmentHandler) CancelAssignment(c *gin.Context) {
	userCtx, exists := middleware.GetUserContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User context not found",
		})
		return
	}

	assignmentIDStr := c.Param("id")
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	// Get the assignment
	assignment, err := h.assignmentRepo.GetAssignmentByID(assignmentID)
	if err != nil || assignment == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Assignment not found",
		})
		return
	}

	// Verify lounge ownership
	owner, err := h.loungeOwnerRepo.GetLoungeOwnerByUserID(userCtx.UserID)
	if err != nil || owner == nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Not a lounge owner",
		})
		return
	}

	lounge, err := h.loungeRepo.GetLoungeByID(assignment.LoungeID)
	if err != nil || lounge == nil || lounge.LoungeOwnerID != owner.ID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You don't own this lounge",
		})
		return
	}

	if err := h.assignmentRepo.CancelAssignment(assignmentID); err != nil {
		log.Printf("ERROR: Failed to cancel assignment: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to cancel assignment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Assignment cancelled successfully",
		"assignment_id": assignmentID,
	})
}
