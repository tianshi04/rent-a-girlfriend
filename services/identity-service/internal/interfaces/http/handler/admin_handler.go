package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/dto"
)

// AdminHandler handles admin-related HTTP requests.
type AdminHandler struct {
	getAccount        *query.GetAccountHandler
	lockAccount       *command.LockAccountHandler
	unlockAccount     *command.UnlockAccountHandler
	approveUpgrade    *command.ApproveUpgradeHandler
	rejectUpgrade     *command.RejectUpgradeHandler
	listUpgradeReqs   *query.ListUpgradeRequestsHandler
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	getAccount *query.GetAccountHandler,
	lockAccount *command.LockAccountHandler,
	unlockAccount *command.UnlockAccountHandler,
	approveUpgrade *command.ApproveUpgradeHandler,
	rejectUpgrade *command.RejectUpgradeHandler,
	listUpgradeReqs *query.ListUpgradeRequestsHandler,
) *AdminHandler {
	return &AdminHandler{
		getAccount:      getAccount,
		lockAccount:     lockAccount,
		unlockAccount:   unlockAccount,
		approveUpgrade:  approveUpgrade,
		rejectUpgrade:   rejectUpgrade,
		listUpgradeReqs: listUpgradeReqs,
	}
}

// GetAccount returns account details.
func (h *AdminHandler) GetAccount(c *gin.Context) {
	account, err := h.getAccount.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AccountResponse{
		ID:             account.ID().String(),
		Email:          account.Email().String(),
		Role:           string(account.Role()),
		Status:         string(account.Status()),
		ViolationCount: account.ViolationCount(),
		CreatedAt:      account.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	})
}

// LockAccount locks a user account.
func (h *AdminHandler) LockAccount(c *gin.Context) {
	var req dto.LockAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetHeader("X-User-Id")
	err := h.lockAccount.Handle(c.Request.Context(), command.LockAccountCommand{
		UserID:  c.Param("id"),
		Reason:  req.Reason,
		AdminID: adminID,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account locked"})
}

// UnlockAccount unlocks a user account.
func (h *AdminHandler) UnlockAccount(c *gin.Context) {
	adminID := c.GetHeader("X-User-Id")
	err := h.unlockAccount.Handle(c.Request.Context(), command.UnlockAccountCommand{
		UserID:  c.Param("id"),
		AdminID: adminID,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account unlocked"})
}

// ApproveUpgrade approves a companion upgrade request.
func (h *AdminHandler) ApproveUpgrade(c *gin.Context) {
	adminID := c.GetHeader("X-User-Id")
	err := h.approveUpgrade.Handle(c.Request.Context(), command.ApproveUpgradeCommand{
		RequestID: c.Param("id"),
		AdminID:   adminID,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "upgrade approved"})
}

// RejectUpgrade rejects a companion upgrade request.
func (h *AdminHandler) RejectUpgrade(c *gin.Context) {
	var req dto.RejectUpgradeRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID := c.GetHeader("X-User-Id")
	err := h.rejectUpgrade.Handle(c.Request.Context(), command.RejectUpgradeCommand{
		RequestID:    c.Param("id"),
		AdminID:      adminID,
		RejectReason: req.Reason,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "upgrade rejected"})
}

// ListUpgradeRequests lists upgrade requests (admin).
func (h *AdminHandler) ListUpgradeRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	requests, total, err := h.listUpgradeReqs.Handle(c.Request.Context(), query.ListUpgradeRequestsQuery{
		Status:   statusPtr,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]dto.UpgradeRequestResponse, 0, len(requests))
	for _, r := range requests {
		item := dto.UpgradeRequestResponse{
			ID:           r.ID().String(),
			UserID:       r.UserID().String(),
			Status:       string(r.Status()),
			Reason:       r.Reason(),
			RejectReason: r.RejectReason(),
			ReviewedBy:   r.ReviewedBy(),
			CreatedAt:    r.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
		if r.ReviewedAt() != nil {
			t := r.ReviewedAt().Format("2006-01-02T15:04:05Z07:00")
			item.ReviewedAt = &t
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, dto.PaginatedResponse{
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
