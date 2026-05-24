package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/util"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

type IdentityGRPCHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	getAccount      *query.GetAccountHandler
	lockAccount     *command.LockAccountHandler
	unlockAccount   *command.UnlockAccountHandler
	approveUpgrade  *command.ApproveUpgradeHandler
	rejectUpgrade   *command.RejectUpgradeHandler
	requestUpgrade  *command.RequestCompanionUpgradeHandler
	listUpgradeReqs *query.ListUpgradeRequestsHandler
	// Auth handlers
	initGoogleAuth *command.InitGoogleAuthHandler
	loginGoogle    *command.LoginGoogleHandler
	refreshToken   *command.RefreshTokenHandler
	logout         *command.LogoutHandler
}

func NewIdentityGRPCHandler(
	getAccount *query.GetAccountHandler,
	lockAccount *command.LockAccountHandler,
	unlockAccount *command.UnlockAccountHandler,
	approveUpgrade *command.ApproveUpgradeHandler,
	rejectUpgrade *command.RejectUpgradeHandler,
	requestUpgrade *command.RequestCompanionUpgradeHandler,
	listUpgradeReqs *query.ListUpgradeRequestsHandler,
	initGoogleAuth *command.InitGoogleAuthHandler,
	loginGoogle *command.LoginGoogleHandler,
	refreshToken *command.RefreshTokenHandler,
	logout *command.LogoutHandler,
) *IdentityGRPCHandler {
	return &IdentityGRPCHandler{
		getAccount:      getAccount,
		lockAccount:     lockAccount,
		unlockAccount:   unlockAccount,
		approveUpgrade:  approveUpgrade,
		rejectUpgrade:   rejectUpgrade,
		requestUpgrade:  requestUpgrade,
		listUpgradeReqs: listUpgradeReqs,
		initGoogleAuth:  initGoogleAuth,
		loginGoogle:     loginGoogle,
		refreshToken:    refreshToken,
		logout:          logout,
	}
}

func (h *IdentityGRPCHandler) GetAccount(ctx context.Context, req *identityv1.GetAccountRequest) (*identityv1.AccountResponse, error) {
	account, err := h.getAccount.Handle(ctx, req.Id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.AccountResponse{
		Id:             account.ID().String(),
		Email:          account.Email().String(),
		Role:           mapRole(account.Role()),
		Status:         mapAccountStatus(account.Status()),
		ViolationCount: int32(account.ViolationCount()),
		CreatedAt:      timestamppb.New(account.CreatedAt()),
	}, nil
}

func (h *IdentityGRPCHandler) ListUpgradeRequests(ctx context.Context, req *identityv1.ListUpgradeRequestsRequest) (*identityv1.ListUpgradeRequestsResponse, error) {
	var statusPtr *string
	if req.Status != identityv1.UpgradeStatus_UPGRADE_STATUS_UNSPECIFIED {
		var s string
		switch req.Status {
		case identityv1.UpgradeStatus_UPGRADE_STATUS_PENDING:
			s = "PENDING"
		case identityv1.UpgradeStatus_UPGRADE_STATUS_APPROVED:
			s = "APPROVED"
		case identityv1.UpgradeStatus_UPGRADE_STATUS_REJECTED:
			s = "REJECTED"
		}
		statusPtr = &s
	}

	requests, total, err := h.listUpgradeReqs.Handle(ctx, query.ListUpgradeRequestsQuery{
		Status:   statusPtr,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	items := make([]*identityv1.UpgradeRequestItem, 0, len(requests))
	for _, r := range requests {
		item := &identityv1.UpgradeRequestItem{
			Id:           r.ID().String(),
			UserId:       r.UserID().String(),
			Status:       mapUpgradeStatus(r.Status()),
			Reason:       r.Reason(),
			RejectReason: r.RejectReason(),
			ReviewedBy:   r.ReviewedBy(),
			CreatedAt:    timestamppb.New(r.CreatedAt()),
		}
		if r.ReviewedAt() != nil {
			item.ReviewedAt = timestamppb.New(*r.ReviewedAt())
		}
		items = append(items, item)
	}

	return &identityv1.ListUpgradeRequestsResponse{
		Data:     items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (h *IdentityGRPCHandler) RequestUpgrade(ctx context.Context, req *identityv1.RequestUpgradeRequest) (*identityv1.MessageResponse, error) {
	userID := util.GetUserID(ctx)

	err := h.requestUpgrade.Handle(ctx, command.RequestCompanionUpgradeCommand{
		UserID: userID,
		Reason: req.Reason,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.MessageResponse{Message: "upgrade request submitted"}, nil
}

func (h *IdentityGRPCHandler) ApproveUpgrade(ctx context.Context, req *identityv1.ApproveUpgradeRequest) (*identityv1.MessageResponse, error) {
	adminID := util.GetUserID(ctx)
	err := h.approveUpgrade.Handle(ctx, command.ApproveUpgradeCommand{
		RequestID: req.Id,
		AdminID:   adminID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &identityv1.MessageResponse{Message: "upgrade approved"}, nil
}

func (h *IdentityGRPCHandler) RejectUpgrade(ctx context.Context, req *identityv1.RejectUpgradeRequest) (*identityv1.MessageResponse, error) {
	adminID := util.GetUserID(ctx)
	err := h.rejectUpgrade.Handle(ctx, command.RejectUpgradeCommand{
		RequestID:    req.Id,
		AdminID:      adminID,
		RejectReason: req.Reason,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &identityv1.MessageResponse{Message: "upgrade rejected"}, nil
}

func (h *IdentityGRPCHandler) LockAccount(ctx context.Context, req *identityv1.LockAccountRequest) (*identityv1.MessageResponse, error) {
	adminID := util.GetUserID(ctx)
	err := h.lockAccount.Handle(ctx, command.LockAccountCommand{
		UserID:  req.Id,
		Reason:  req.Reason,
		AdminID: adminID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &identityv1.MessageResponse{Message: "account locked"}, nil
}

func (h *IdentityGRPCHandler) UnlockAccount(ctx context.Context, req *identityv1.UnlockAccountRequest) (*identityv1.MessageResponse, error) {
	adminID := util.GetUserID(ctx)
	err := h.unlockAccount.Handle(ctx, command.UnlockAccountCommand{
		UserID:  req.Id,
		AdminID: adminID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &identityv1.MessageResponse{Message: "account unlocked"}, nil
}

// Auth Methods

func (h *IdentityGRPCHandler) InitGoogleAuth(ctx context.Context, _ *identityv1.InitGoogleAuthRequest) (*identityv1.InitGoogleAuthResponse, error) {
	result, err := h.initGoogleAuth.Handle(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.InitGoogleAuthResponse{
		AuthUrl:       result.AuthURL,
		State:         result.State,
		CodeChallenge: result.CodeChallenge,
	}, nil
}

func (h *IdentityGRPCHandler) LoginGoogle(ctx context.Context, req *identityv1.LoginGoogleRequest) (*identityv1.TokenResponse, error) {
	tokenPair, err := h.loginGoogle.Handle(ctx, command.LoginGoogleCommand{
		Code:  req.Code,
		State: req.State,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (h *IdentityGRPCHandler) RefreshToken(ctx context.Context, req *identityv1.RefreshTokenRequest) (*identityv1.TokenResponse, error) {
	tokenPair, err := h.refreshToken.Handle(ctx, command.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (h *IdentityGRPCHandler) Logout(ctx context.Context, req *identityv1.LogoutRequest) (*identityv1.MessageResponse, error) {
	err := h.logout.Handle(ctx, command.LogoutCommand{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &identityv1.MessageResponse{Message: "logged out"}, nil
}

func mapDomainError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case domainerr.ErrAccountNotFound:
		return status.Error(codes.NotFound, err.Error())
	case domainerr.ErrAccountLocked:
		return status.Error(codes.PermissionDenied, err.Error())
	case domainerr.ErrInvalidRefreshToken, domainerr.ErrRefreshTokenReuse, domainerr.ErrInvalidOAuthToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case domainerr.ErrInvalidEmail, domainerr.ErrInvalidRole, domainerr.ErrPKCEStateNotFound:
		return status.Error(codes.InvalidArgument, err.Error())
	case domainerr.ErrEmailAlreadyExists, domainerr.ErrConcurrencyConflict, domainerr.ErrAlreadyCompanion, domainerr.ErrUpgradeRequestPending:
		return status.Error(codes.AlreadyExists, err.Error())
	case domainerr.ErrUpgradeRequestNotFound:
		return status.Error(codes.NotFound, err.Error())
	case domainerr.ErrInvalidUpgradeStatus, domainerr.ErrNotClient:
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func mapRole(r vo.Role) identityv1.AccountRole {
	switch r {
	case vo.RoleClient:
		return identityv1.AccountRole_ACCOUNT_ROLE_CLIENT
	case vo.RoleCompanion:
		return identityv1.AccountRole_ACCOUNT_ROLE_COMPANION
	case vo.RoleAdmin:
		return identityv1.AccountRole_ACCOUNT_ROLE_ADMIN
	default:
		return identityv1.AccountRole_ACCOUNT_ROLE_UNSPECIFIED
	}
}

func mapAccountStatus(s vo.AccountStatus) identityv1.AccountStatus {
	switch s {
	case vo.StatusActive:
		return identityv1.AccountStatus_ACCOUNT_STATUS_ACTIVE
	case vo.StatusLocked:
		return identityv1.AccountStatus_ACCOUNT_STATUS_LOCKED
	default:
		return identityv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED
	}
}

func mapUpgradeStatus(s vo.UpgradeStatus) identityv1.UpgradeStatus {
	switch s {
	case vo.UpgradeStatusPending:
		return identityv1.UpgradeStatus_UPGRADE_STATUS_PENDING
	case vo.UpgradeStatusApproved:
		return identityv1.UpgradeStatus_UPGRADE_STATUS_APPROVED
	case vo.UpgradeStatusRejected:
		return identityv1.UpgradeStatus_UPGRADE_STATUS_REJECTED
	default:
		return identityv1.UpgradeStatus_UPGRADE_STATUS_UNSPECIFIED
	}
}
