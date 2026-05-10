package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/rent-a-girlfriend/identity-service/api/proto"
	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/util"
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
		return nil, err
	}

	return &identityv1.AccountResponse{
		Id:             account.ID().String(),
		Email:          account.Email().String(),
		Role:           string(account.Role()),
		Status:         string(account.Status()),
		ViolationCount: int32(account.ViolationCount()),
		CreatedAt:      timestamppb.New(account.CreatedAt()),
	}, nil
}

func (h *IdentityGRPCHandler) ListUpgradeRequests(ctx context.Context, req *identityv1.ListUpgradeRequestsRequest) (*identityv1.ListUpgradeRequestsResponse, error) {
	var statusPtr *string
	if req.Status != "" {
		statusPtr = &req.Status
	}

	requests, total, err := h.listUpgradeReqs.Handle(ctx, query.ListUpgradeRequestsQuery{
		Status:   statusPtr,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*identityv1.UpgradeRequestItem, 0, len(requests))
	for _, r := range requests {
		item := &identityv1.UpgradeRequestItem{
			Id:           r.ID().String(),
			UserId:       r.UserID().String(),
			Status:       string(r.Status()),
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
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}
	return &identityv1.MessageResponse{Message: "account unlocked"}, nil
}

// Auth Methods

func (h *IdentityGRPCHandler) InitGoogleAuth(ctx context.Context, _ *identityv1.InitGoogleAuthRequest) (*identityv1.InitGoogleAuthResponse, error) {
	result, err := h.initGoogleAuth.Handle(ctx)
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	return &identityv1.MessageResponse{Message: "logged out"}, nil
}
