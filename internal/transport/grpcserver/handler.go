package grpcserver

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	serversv1 "github.com/MrEsbens/messenger-servers-service/api/servers/v1"
	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	"github.com/MrEsbens/messenger-servers-service/internal/service"
	"github.com/google/uuid"
)

// Handler — gRPC handler для Servers Service
type Handler struct {
	serversv1.UnimplementedServersServiceServer
	serverService service.ServerServiceInterface
}

func NewHandler(serverService service.ServerServiceInterface) *Handler {
	return &Handler{
		serverService: serverService,
	}
}

// ───────────────────────────────────────────────────────────
// Helpers: unpack optional proto fields
// ───────────────────────────────────────────────────────────

func unpackInt32(ptr *int32, defaultValue int) int {
	if ptr != nil {
		return int(*ptr)
	}
	return defaultValue
}

func unpackString(ptr *string, defaultValue string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

func unpackBool(ptr *bool, defaultValue bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

func unpackModerationAction(ptr *string) domain.ModerationAction {
	if ptr != nil && *ptr != "" {
		return domain.ModerationAction(*ptr)
	}
	return domain.ActionNone
}

// ───────────────────────────────────────────────────────────
// Server CRUD
// ───────────────────────────────────────────────────────────

func (h *Handler) CreateServer(ctx context.Context, req *serversv1.CreateServerRequest) (*serversv1.CreateServerResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
	}

	server, err := h.serverService.CreateServer(ctx, req.Name, ownerID, req.Description)
	if err != nil {
		if err.Error() == "owner not found" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create server: %v", err))
	}

	return &serversv1.CreateServerResponse{
		ServerId:  server.ID.String(),
		CreatedAt: timestamppb.New(server.CreatedAt),
	}, nil
}

func (h *Handler) GetServer(ctx context.Context, req *serversv1.GetServerRequest) (*serversv1.ServerDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester_id")
	}

	server, err := h.serverService.GetServer(ctx, serverID, requesterID)
	if err != nil {
		if err == domain.ErrServerNotFound || err == domain.ErrNotMember {
			return nil, status.Error(codes.NotFound, "server not found or access denied")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get server: %v", err))
	}

	return serverToDTO(server), nil
}

func (h *Handler) UpdateServer(ctx context.Context, req *serversv1.UpdateServerRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	updaterID, err := uuid.Parse(req.UpdaterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updater_id")
	}

	err = h.serverService.UpdateServer(ctx, serverID, updaterID, req.Name, req.Description)
	if err != nil {
		if err == domain.ErrServerNotFound {
			return nil, status.Error(codes.NotFound, "server not found")
		}
		if err == domain.ErrNotServerOwner {
			return nil, status.Error(codes.PermissionDenied, "only server owner can update")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update server: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) DeleteServer(ctx context.Context, req *serversv1.DeleteServerRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	deleterID, err := uuid.Parse(req.DeleterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deleter_id")
	}

	err = h.serverService.DeleteServer(ctx, serverID, deleterID)
	if err != nil {
		if err == domain.ErrServerNotFound {
			return nil, status.Error(codes.NotFound, "server not found")
		}
		if err == domain.ErrNotServerOwner {
			return nil, status.Error(codes.PermissionDenied, "only server owner can delete")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete server: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListUserServers(ctx context.Context, req *serversv1.ListUserServersRequest) (*serversv1.ListUserServersResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	servers, err := h.serverService.ListUserServers(ctx, userID, limit, int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list servers: %v", err))
	}

	dtoServers := make([]*serversv1.ServerDTO, 0, len(servers))
	for _, s := range servers {
		dtoServers = append(dtoServers, serverToDTO(s))
	}

	return &serversv1.ListUserServersResponse{
		Servers: dtoServers,
	}, nil
}

// ───────────────────────────────────────────────────────────
// Configs
// ───────────────────────────────────────────────────────────

func (h *Handler) GetServerConfig(ctx context.Context, req *serversv1.GetServerConfigRequest) (*serversv1.ServerConfigDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	config, err := h.serverService.GetServerConfig(ctx, serverID)
	if err != nil {
		if err == domain.ErrConfigNotFound {
			return nil, status.Error(codes.NotFound, "config not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get config: %v", err))
	}

	return serverConfigToDTO(config), nil
}

func (h *Handler) UpdateServerConfig(ctx context.Context, req *serversv1.UpdateServerConfigRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	updaterID, err := uuid.Parse(req.UpdaterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updater_id")
	}

	config := &domain.ServerConfig{
		ServerID:                serverID,
		MaxMembers:              unpackInt32(req.MaxMembers, 500),
		MaxChannels:             unpackInt32(req.MaxChannels, 500),
		DefaultNotificationMode: unpackString(req.DefaultNotificationMode, "all"),
		ModerationEnabled:       unpackBool(req.ModerationEnabled, false),
		UpdatedAt:               time.Now(),
	}

	err = h.serverService.UpdateServerConfig(ctx, serverID, updaterID, config)
	if err != nil {
		if err == domain.ErrServerNotFound {
			return nil, status.Error(codes.NotFound, "server not found")
		}
		if err == domain.ErrNotServerOwner {
			return nil, status.Error(codes.PermissionDenied, "only server owner can update config")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update config: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetModerationConfig(ctx context.Context, req *serversv1.GetModerationConfigRequest) (*serversv1.ModerationConfigDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	config, err := h.serverService.GetModerationConfig(ctx, serverID)
	if err != nil {
		if err == domain.ErrModerationConfigNotFound {
			return nil, status.Error(codes.NotFound, "moderation config not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get moderation config: %v", err))
	}

	return moderationConfigToDTO(config), nil
}

func (h *Handler) UpdateModerationConfig(ctx context.Context, req *serversv1.UpdateModerationConfigRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	updaterID, err := uuid.Parse(req.UpdaterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updater_id")
	}

	// 🔧 Правильно конвертируем *string → domain.ModerationAction
	config := &domain.ModerationConfig{
		ServerID:               serverID,
		ProfanityFilterAction:  unpackModerationAction(req.ProfanityFilterAction),
		ToxicityFilterAction:   unpackModerationAction(req.ToxicityFilterAction),
		NsfwTextFilterAction:   unpackModerationAction(req.NsfwTextFilterAction),
		PoliticalFilterAction:  unpackModerationAction(req.PoliticalFilterAction),
		HateSpeechFilterAction: unpackModerationAction(req.HateSpeechFilterAction),
		UpdatedAt:              time.Now(),
	}

	err = h.serverService.UpdateModerationConfig(ctx, serverID, updaterID, config)
	if err != nil {
		if err == domain.ErrServerNotFound {
			return nil, status.Error(codes.NotFound, "server not found")
		}
		if err == domain.ErrNotServerOwner {
			return nil, status.Error(codes.PermissionDenied, "only server owner can update moderation config")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update moderation config: %v", err))
	}

	return &emptypb.Empty{}, nil
}

// ───────────────────────────────────────────────────────────
// Members
// ───────────────────────────────────────────────────────────

func (h *Handler) AddMember(ctx context.Context, req *serversv1.AddMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	addedBy, err := uuid.Parse(req.AddedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid added_by")
	}

	err = h.serverService.AddMember(ctx, serverID, userID, addedBy)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this server")
		}
		if err == domain.ErrAlreadyMember {
			return nil, status.Error(codes.AlreadyExists, "user is already a member")
		}
		if err == domain.ErrMaxMembersReached {
			return nil, status.Error(codes.ResourceExhausted, "max members reached")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to add member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *serversv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	removedBy, err := uuid.Parse(req.RemovedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid removed_by")
	}

	err = h.serverService.RemoveMember(ctx, serverID, userID, removedBy)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.NotFound, "user is not a member")
		}
		if err == domain.ErrCannotKickOwner {
			return nil, status.Error(codes.PermissionDenied, "cannot kick server owner")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetMember(ctx context.Context, req *serversv1.GetMemberRequest) (*serversv1.MemberDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	member, err := h.serverService.GetMember(ctx, serverID, userID)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.NotFound, "user is not a member")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get member: %v", err))
	}

	return memberToDTO(member), nil
}

func (h *Handler) ListMembers(ctx context.Context, req *serversv1.ListMembersRequest) (*serversv1.ListMembersResponse, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	members, err := h.serverService.ListMembers(ctx, serverID, limit, int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list members: %v", err))
	}

	dtoMembers := make([]*serversv1.MemberDTO, 0, len(members))
	for _, m := range members {
		dtoMembers = append(dtoMembers, memberToDTO(m))
	}

	return &serversv1.ListMembersResponse{
		Members: dtoMembers,
	}, nil
}

func (h *Handler) BanMember(ctx context.Context, req *serversv1.BanMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	bannedBy, err := uuid.Parse(req.BannedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid banned_by")
	}

	err = h.serverService.BanMember(ctx, serverID, userID, bannedBy)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.NotFound, "user is not a member")
		}
		if err == domain.ErrCannotBanOwner {
			return nil, status.Error(codes.PermissionDenied, "cannot ban server owner")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to ban member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) UnbanMember(ctx context.Context, req *serversv1.UnbanMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	unbannedBy, err := uuid.Parse(req.UnbannedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid unbanned_by")
	}

	err = h.serverService.UnbanMember(ctx, serverID, userID, unbannedBy)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unban member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) MuteMember(ctx context.Context, req *serversv1.MuteMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	mutedBy, err := uuid.Parse(req.MutedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid muted_by")
	}

	var duration *time.Duration
	if req.DurationHours > 0 {
		d := time.Duration(req.DurationHours) * time.Hour
		duration = &d
	}

	// 🔴 Передаём mutedBy в сервис
	err = h.serverService.MuteMember(ctx, serverID, userID, mutedBy, duration)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this server")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mute member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) UnmuteMember(ctx context.Context, req *serversv1.UnmuteMemberRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	unmutedBy, err := uuid.Parse(req.UnmutedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid unmuted_by")
	}

	err = h.serverService.UnmuteMember(ctx, serverID, userID, unmutedBy)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this server")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmute member: %v", err))
	}

	return &emptypb.Empty{}, nil
}

// ───────────────────────────────────────────────────────────
// Roles
// ───────────────────────────────────────────────────────────

func (h *Handler) CreateRole(ctx context.Context, req *serversv1.CreateRoleRequest) (*serversv1.RoleDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	creatorID, err := uuid.Parse(req.CreatorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid creator_id")
	}

	role, err := h.serverService.CreateRole(ctx, serverID, req.Name, creatorID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create role: %v", err))
	}

	return roleToDTO(role), nil
}

func (h *Handler) UpdateRole(ctx context.Context, req *serversv1.UpdateRoleRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}

	updaterID, err := uuid.Parse(req.UpdaterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updater_id")
	}

	err = h.serverService.UpdateRole(ctx, serverID, roleID, updaterID, req.Name, req.Color, req.Permissions)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update role: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) DeleteRole(ctx context.Context, req *serversv1.DeleteRoleRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}

	deleterID, err := uuid.Parse(req.DeleterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deleter_id")
	}

	err = h.serverService.DeleteRole(ctx, serverID, roleID, deleterID)
	if err != nil {
		if err == domain.ErrCannotDeleteDefaultRole {
			return nil, status.Error(codes.PermissionDenied, "cannot delete default role")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete role: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) AssignRole(ctx context.Context, req *serversv1.AssignRoleRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	memberID, err := uuid.Parse(req.MemberId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid member_id")
	}

	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}

	assignedBy, err := uuid.Parse(req.AssignedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid assigned_by")
	}

	err = h.serverService.AssignRole(ctx, serverID, memberID, roleID, assignedBy)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to assign role: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveRole(ctx context.Context, req *serversv1.RemoveRoleRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	memberID, err := uuid.Parse(req.MemberId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid member_id")
	}

	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}

	removedBy, err := uuid.Parse(req.RemovedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid removed_by")
	}

	err = h.serverService.RemoveRole(ctx, serverID, memberID, roleID, removedBy)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove role: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetMemberRoles(ctx context.Context, req *serversv1.GetMemberRolesRequest) (*serversv1.GetMemberRolesResponse, error) {
	memberID, err := uuid.Parse(req.MemberId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid member_id")
	}

	roles, err := h.serverService.GetMemberRoles(ctx, memberID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get member roles: %v", err))
	}

	dtoRoles := make([]*serversv1.RoleDTO, 0, len(roles))
	for _, r := range roles {
		dtoRoles = append(dtoRoles, roleToDTO(r))
	}

	return &serversv1.GetMemberRolesResponse{
		Roles: dtoRoles,
	}, nil
}

// ───────────────────────────────────────────────────────────
// Server Chats
// ───────────────────────────────────────────────────────────

func (h *Handler) CreateServerChat(ctx context.Context, req *serversv1.CreateServerChatRequest) (*serversv1.ServerChatDTO, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	chat, err := h.serverService.CreateServerChat(ctx, serverID, req.Name, createdBy)
	if err != nil {
		if err == domain.ErrNotMember {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this server")
		}
		if err == domain.ErrMaxChannelsReached {
			return nil, status.Error(codes.ResourceExhausted, "max channels reached")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create server chat: %v", err))
	}

	return serverChatToDTO(chat), nil
}

func (h *Handler) ListServerChats(ctx context.Context, req *serversv1.ListServerChatsRequest) (*serversv1.ListServerChatsResponse, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	chats, err := h.serverService.ListServerChats(ctx, serverID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list server chats: %v", err))
	}

	dtoChats := make([]*serversv1.ServerChatDTO, 0, len(chats))
	for _, c := range chats {
		dtoChats = append(dtoChats, serverChatToDTO(c))
	}

	return &serversv1.ListServerChatsResponse{
		Chats: dtoChats,
	}, nil
}

func (h *Handler) DeleteServerChat(ctx context.Context, req *serversv1.DeleteServerChatRequest) (*emptypb.Empty, error) {
	serverID, err := uuid.Parse(req.ServerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid server_id")
	}

	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}

	deleterID, err := uuid.Parse(req.DeleterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deleter_id")
	}

	err = h.serverService.DeleteServerChat(ctx, serverID, chatID, deleterID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete server chat: %v", err))
	}

	return &emptypb.Empty{}, nil
}

// ───────────────────────────────────────────────────────────
// Helpers: Domain → DTO
// ───────────────────────────────────────────────────────────

func serverToDTO(s *domain.Server) *serversv1.ServerDTO {
	return &serversv1.ServerDTO{
		Id:          s.ID.String(),
		Name:        s.Name,
		Description: s.Description,
		OwnerId:     s.OwnerID.String(),
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
		IsDeleted:   s.DeletedAt != nil,
	}
}

func serverConfigToDTO(c *domain.ServerConfig) *serversv1.ServerConfigDTO {
	return &serversv1.ServerConfigDTO{
		ServerId:                c.ServerID.String(),
		MaxMembers:              int32(c.MaxMembers),
		MaxChannels:             int32(c.MaxChannels),
		DefaultNotificationMode: c.DefaultNotificationMode,
		ModerationEnabled:       c.ModerationEnabled,
	}
}

func moderationConfigToDTO(c *domain.ModerationConfig) *serversv1.ModerationConfigDTO {
	return &serversv1.ModerationConfigDTO{
		ServerId:               c.ServerID.String(),
		ProfanityFilterAction:  string(c.ProfanityFilterAction),
		ToxicityFilterAction:   string(c.ToxicityFilterAction),
		NsfwTextFilterAction:   string(c.NsfwTextFilterAction),
		PoliticalFilterAction:  string(c.PoliticalFilterAction),
		HateSpeechFilterAction: string(c.HateSpeechFilterAction),
	}
}

func memberToDTO(m *domain.ServerMember) *serversv1.MemberDTO {
	dto := &serversv1.MemberDTO{
		UserId:   m.UserID.String(),
		ServerId: m.ServerID.String(),
		JoinedAt: timestamppb.New(m.JoinedAt),
		IsMuted:  m.IsMuted,
		IsBanned: m.IsBanned,
		RoleIds:  []string{},
	}

	if m.MutedUntil != nil {
		dto.MutedUntil = timestamppb.New(*m.MutedUntil)
	}

	return dto
}

func roleToDTO(r *domain.ServerRole) *serversv1.RoleDTO {
	return &serversv1.RoleDTO{
		Id:          r.ID.String(),
		ServerId:    r.ServerID.String(),
		Name:        r.Name,
		Color:       r.Color,
		Permissions: r.Permissions,
		Position:    int32(r.Position),
		IsDefault:   r.IsDefault,
	}
}

func serverChatToDTO(c *repository.ServerChat) *serversv1.ServerChatDTO {
	return &serversv1.ServerChatDTO{
		ChatId:    c.ChatID.String(),
		ServerId:  c.ServerID.String(),
		Name:      c.Name,
		Position:  int32(c.Position),
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}
