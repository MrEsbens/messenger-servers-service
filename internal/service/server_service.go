package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	"github.com/MrEsbens/messenger-servers-service/internal/repository/redis"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcclient"
	"github.com/google/uuid"
)

type ServerServiceInterface interface {
	// Server CRUD
	CreateServer(ctx context.Context, name string, ownerID uuid.UUID, description *string) (*domain.Server, error)
	GetServer(ctx context.Context, serverID, requesterID uuid.UUID) (*domain.Server, error)
	UpdateServer(ctx context.Context, serverID, updaterID uuid.UUID, name *string, description *string) error
	DeleteServer(ctx context.Context, serverID, deleterID uuid.UUID) error
	ListUserServers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Server, error)

	// Configs
	GetServerConfig(ctx context.Context, serverID uuid.UUID) (*domain.ServerConfig, error)
	UpdateServerConfig(ctx context.Context, serverID, updaterID uuid.UUID, config *domain.ServerConfig) error
	GetModerationConfig(ctx context.Context, serverID uuid.UUID) (*domain.ModerationConfig, error)
	UpdateModerationConfig(ctx context.Context, serverID, updaterID uuid.UUID, config *domain.ModerationConfig) error

	// Members
	AddMember(ctx context.Context, serverID, userID, addedBy uuid.UUID) error
	RemoveMember(ctx context.Context, serverID, userID, removedBy uuid.UUID) error
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*domain.ServerMember, error)
	ListMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error)
	BanMember(ctx context.Context, serverID, userID, bannedBy uuid.UUID) error
	UnbanMember(ctx context.Context, serverID, userID, unbannedBy uuid.UUID) error
	MuteMember(ctx context.Context, serverID, userID, mutedBy uuid.UUID, duration *time.Duration) error
	UnmuteMember(ctx context.Context, serverID, userID, unmutedBy uuid.UUID) error

	// Roles
	CreateRole(ctx context.Context, serverID uuid.UUID, name string, creatorID uuid.UUID) (*domain.ServerRole, error)
	UpdateRole(ctx context.Context, serverID, roleID, updaterID uuid.UUID, name *string, color *string, permissions []string) error
	DeleteRole(ctx context.Context, serverID, roleID, deleterID uuid.UUID) error
	AssignRole(ctx context.Context, serverID, memberID, roleID uuid.UUID, assignedBy uuid.UUID) error
	RemoveRole(ctx context.Context, serverID, memberID, roleID uuid.UUID, removedBy uuid.UUID) error
	GetMemberRoles(ctx context.Context, memberID uuid.UUID) ([]*domain.ServerRole, error)

	// Server Chats
	CreateServerChat(ctx context.Context, serverID uuid.UUID, name string, createdBy uuid.UUID) (*repository.ServerChat, error)
	ListServerChats(ctx context.Context, serverID, requesterID uuid.UUID) ([]*repository.ServerChat, error)
	DeleteServerChat(ctx context.Context, serverID, chatID, deleterID uuid.UUID) error
}

type ServerService struct {
	serverRepo     repository.ServerRepository
	configRepo     repository.ConfigRepository
	memberRepo     repository.MemberRepository
	roleRepo       repository.RoleRepository
	chatRepo       repository.ChatRepository
	identityClient grpcclient.IdentityClientInterface
	chatsClient    grpcclient.ChatsClientInterface
	cacheRepo      redis.CacheRepository
}

// NewServerService создаёт новый экземпляр сервиса.
func NewServerService(
	serverRepo repository.ServerRepository,
	configRepo repository.ConfigRepository,
	memberRepo repository.MemberRepository,
	roleRepo repository.RoleRepository,
	chatRepo repository.ChatRepository,
	identityClient grpcclient.IdentityClientInterface,
	chatsClient grpcclient.ChatsClientInterface,
	cacheRepo redis.CacheRepository,
) ServerServiceInterface {
	return &ServerService{
		serverRepo:     serverRepo,
		configRepo:     configRepo,
		memberRepo:     memberRepo,
		roleRepo:       roleRepo,
		chatRepo:       chatRepo,
		identityClient: identityClient,
		chatsClient:    chatsClient,
		cacheRepo:      cacheRepo,
	}
}

// ───────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────

func (s *ServerService) getUserPermissions(ctx context.Context, serverID, userID uuid.UUID) ([]domain.Permission, error) {
	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, domain.ErrNotMember
		}
		return nil, err
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	permissionSet := make(map[domain.Permission]bool)
	for _, role := range roles {
		for _, perm := range role.Permissions {
			permissionSet[domain.Permission(perm)] = true
		}
	}

	permissions := make([]domain.Permission, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

func (s *ServerService) hasPermission(ctx context.Context, serverID, userID uuid.UUID, requiredPermissions ...domain.Permission) (bool, error) {
	permissions, err := s.getUserPermissions(ctx, serverID, userID)
	if err != nil {
		return false, err
	}

	for _, required := range requiredPermissions {
		for _, has := range permissions {
			if has == required {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *ServerService) isServerAdmin(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return false, err
	}

	if server.OwnerID == userID {
		return true, nil
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return false, err
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return false, err
	}

	return domain.HasManageServerAccess(roles), nil
}

func (s *ServerService) canManageChannels(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return false, err
	}

	if server.OwnerID == userID {
		return true, nil
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return false, err
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return false, err
	}

	return domain.HasManageChannelsAccess(roles), nil
}

func (s *ServerService) checkMemberAccess(ctx context.Context, serverID, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.ErrNotMember
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	if member.IsBanned {
		return domain.ErrMemberIsBanned
	}

	if member.IsMuted && !member.IsMuteExpired() {
		return domain.ErrMemberIsMuted
	}

	return nil
}

// ───────────────────────────────────────────────────────────
// Server CRUD
// ───────────────────────────────────────────────────────────

func (s *ServerService) CreateServer(ctx context.Context, name string, ownerID uuid.UUID, description *string) (*domain.Server, error) {
	if s.identityClient == nil {
		return nil, fmt.Errorf("identity service client not initialized")
	}

	exists, err := s.identityClient.UserExists(ctx, ownerID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to verify owner: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("owner not found: %s", ownerID)
	}

	server, err := domain.NewServer(name, ownerID, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create server domain object: %w", err)
	}

	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	member, err := domain.NewServerMember(server.ID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to create owner member: %w", err)
	}
	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return server, nil
}

func (s *ServerService) GetServer(ctx context.Context, serverID, requesterID uuid.UUID) (*domain.Server, error) {
	if err := s.checkMemberAccess(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, domain.ErrServerNotFound
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return server, nil
}

func (s *ServerService) UpdateServer(ctx context.Context, serverID, updaterID uuid.UUID, name *string, description *string) error {
	isAdmin, err := s.isServerAdmin(ctx, serverID, updaterID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return domain.ErrPermissionDenied
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrServerNotFound
		}
		return err
	}

	server.Update(name, description)
	return s.serverRepo.Update(ctx, server)
}

func (s *ServerService) DeleteServer(ctx context.Context, serverID, deleterID uuid.UUID) error {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrServerNotFound
		}
		return err
	}

	if server.OwnerID != deleterID {
		return domain.ErrNotServerOwner
	}

	return s.serverRepo.SoftDelete(ctx, serverID)
}

func (s *ServerService) ListUserServers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Server, error) {
	members, err := s.memberRepo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user members: %w", err)
	}

	servers := make([]*domain.Server, 0, len(members))
	for _, member := range members {
		if member.IsBanned {
			continue
		}
		server, err := s.serverRepo.GetByID(ctx, member.ServerID)
		if err != nil {
			continue
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// ───────────────────────────────────────────────────────────
// Configs
// ───────────────────────────────────────────────────────────

func (s *ServerService) GetServerConfig(ctx context.Context, serverID uuid.UUID) (*domain.ServerConfig, error) {
	config, err := s.configRepo.GetServerConfig(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get server config: %w", err)
	}
	return config, nil
}

func (s *ServerService) UpdateServerConfig(ctx context.Context, serverID, updaterID uuid.UUID, config *domain.ServerConfig) error {
	isAdmin, err := s.isServerAdmin(ctx, serverID, updaterID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return domain.ErrPermissionDenied
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return s.configRepo.UpdateServerConfig(ctx, config)
}

func (s *ServerService) GetModerationConfig(ctx context.Context, serverID uuid.UUID) (*domain.ModerationConfig, error) {
	config, err := s.configRepo.GetModerationConfig(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, domain.ErrModerationConfigNotFound
		}
		return nil, fmt.Errorf("failed to get moderation config: %w", err)
	}
	return config, nil
}

func (s *ServerService) UpdateModerationConfig(ctx context.Context, serverID, updaterID uuid.UUID, config *domain.ModerationConfig) error {
	isAdmin, err := s.isServerAdmin(ctx, serverID, updaterID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return domain.ErrPermissionDenied
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid moderation config: %w", err)
	}

	if err := s.configRepo.UpdateModerationConfig(ctx, config); err != nil {
		return err
	}

	if s.cacheRepo != nil {
		if err := s.cacheRepo.InvalidateModerationConfig(ctx, serverID); err != nil {
			return fmt.Errorf("Failed to invalidate moderation config cache: %w", err)
		}
	}

	return nil
}

// ───────────────────────────────────────────────────────────
// Members
// ───────────────────────────────────────────────────────────

func (s *ServerService) AddMember(ctx context.Context, serverID, userID, addedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, addedBy); err != nil {
		return err
	}

	if s.identityClient == nil {
		return fmt.Errorf("identity service client not initialized")
	}

	exists, err := s.identityClient.UserExists(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	exists, err = s.memberRepo.Exists(ctx, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to check member exists: %w", err)
	}
	if exists {
		return domain.ErrAlreadyMember
	}

	config, err := s.configRepo.GetServerConfig(ctx, serverID)
	if err != nil {
		return err
	}

	count, err := s.memberRepo.CountByServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}
	if count >= config.MaxMembers {
		return domain.ErrMaxMembersReached
	}

	member, err := domain.NewServerMember(serverID, userID)
	if err != nil {
		return err
	}
	return s.memberRepo.Create(ctx, member)
}

func (s *ServerService) RemoveMember(ctx context.Context, serverID, userID, removedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, removedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != removedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, removedBy, domain.PermKickMembers)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	if member.UserID == server.OwnerID {
		return domain.ErrCannotKickOwner
	}

	return s.memberRepo.Delete(ctx, member.ID)
}

func (s *ServerService) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*domain.ServerMember, error) {
	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, domain.ErrNotMember
		}
		return nil, err
	}
	return member, nil
}

func (s *ServerService) ListMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error) {
	return s.memberRepo.GetByServer(ctx, serverID, limit, offset)
}

func (s *ServerService) BanMember(ctx context.Context, serverID, userID, bannedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, bannedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != bannedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, bannedBy, domain.PermBanMembers)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	targetMember, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	if targetMember.UserID == server.OwnerID {
		return domain.ErrCannotBanOwner
	}

	targetMember.Ban()
	return s.memberRepo.Update(ctx, targetMember)
}

func (s *ServerService) UnbanMember(ctx context.Context, serverID, userID, unbannedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, unbannedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != unbannedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, unbannedBy, domain.PermBanMembers)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return err
	}

	member.Unban()
	return s.memberRepo.Update(ctx, member)
}

func (s *ServerService) MuteMember(ctx context.Context, serverID, userID, mutedBy uuid.UUID, duration *time.Duration) error {
	if err := s.checkMemberAccess(ctx, serverID, mutedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != mutedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, mutedBy, domain.PermMuteMembers)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return err
	}

	if member.UserID == server.OwnerID {
		return domain.ErrCannotBanOwner
	}

	member.Mute(duration)
	return s.memberRepo.Update(ctx, member)
}

func (s *ServerService) UnmuteMember(ctx context.Context, serverID, userID, unmutedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, unmutedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != unmutedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, unmutedBy, domain.PermMuteMembers)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return err
	}

	member.Unmute()
	return s.memberRepo.Update(ctx, member)
}

// ───────────────────────────────────────────────────────────
// Roles
// ───────────────────────────────────────────────────────────

func (s *ServerService) CreateRole(ctx context.Context, serverID uuid.UUID, name string, creatorID uuid.UUID) (*domain.ServerRole, error) {
	if err := s.checkMemberAccess(ctx, serverID, creatorID); err != nil {
		return nil, err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if server.OwnerID != creatorID {
		hasPerm, err := s.hasPermission(ctx, serverID, creatorID, domain.PermManageRoles)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, domain.ErrPermissionDenied
		}
	}

	role, err := domain.NewServerRole(serverID, name, false)
	if err != nil {
		return nil, err
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

func (s *ServerService) UpdateRole(ctx context.Context, serverID, roleID, updaterID uuid.UUID, name *string, color *string, permissions []string) error {
	if err := s.checkMemberAccess(ctx, serverID, updaterID); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != updaterID {
		hasPerm, err := s.hasPermission(ctx, serverID, updaterID, domain.PermManageRoles)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	role.Update(name, color, permissions, nil)
	return s.roleRepo.Update(ctx, role)
}

func (s *ServerService) DeleteRole(ctx context.Context, serverID, roleID, deleterID uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, deleterID); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != deleterID {
		hasPerm, err := s.hasPermission(ctx, serverID, deleterID, domain.PermManageRoles)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsDefault {
		return domain.ErrCannotDeleteDefaultRole
	}

	return s.roleRepo.Delete(ctx, roleID)
}

func (s *ServerService) AssignRole(ctx context.Context, serverID, memberID, roleID uuid.UUID, assignedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, assignedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != assignedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, assignedBy, domain.PermManageRoles)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	if server.OwnerID != assignedBy {
		assignedByMember, err := s.memberRepo.GetByServerAndUser(ctx, serverID, assignedBy)
		if err != nil {
			return err
		}

		assignedByRoles, err := s.roleRepo.GetMemberRoles(ctx, assignedByMember.ID)
		if err != nil {
			return err
		}

		maxPosition := 0
		for _, r := range assignedByRoles {
			if r.Position > maxPosition {
				maxPosition = r.Position
			}
		}

		if role.Position > maxPosition {
			return domain.ErrPermissionDenied
		}
	}

	return s.roleRepo.AssignToMember(ctx, memberID, roleID)
}

func (s *ServerService) RemoveRole(ctx context.Context, serverID, memberID, roleID uuid.UUID, removedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, removedBy); err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if server.OwnerID != removedBy {
		hasPerm, err := s.hasPermission(ctx, serverID, removedBy, domain.PermManageRoles)
		if err != nil {
			return err
		}
		if !hasPerm {
			return domain.ErrPermissionDenied
		}
	}

	return s.roleRepo.RemoveFromMember(ctx, memberID, roleID)
}

func (s *ServerService) GetMemberRoles(ctx context.Context, memberID uuid.UUID) ([]*domain.ServerRole, error) {
	return s.roleRepo.GetMemberRoles(ctx, memberID)
}

// ───────────────────────────────────────────────────────────
// Server Chats
// ───────────────────────────────────────────────────────────

func (s *ServerService) CreateServerChat(ctx context.Context, serverID uuid.UUID, name string, createdBy uuid.UUID) (*repository.ServerChat, error) {
	if err := s.checkMemberAccess(ctx, serverID, createdBy); err != nil {
		return nil, err
	}

	canManage, err := s.canManageChannels(ctx, serverID, createdBy)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, domain.ErrPermissionDenied
	}

	config, err := s.configRepo.GetServerConfig(ctx, serverID)
	if err != nil {
		return nil, err
	}

	chats, err := s.chatRepo.GetByServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if len(chats) >= config.MaxChannels {
		return nil, domain.ErrMaxChannelsReached
	}

	if s.chatsClient == nil {
		return nil, fmt.Errorf("chats service client not initialized")
	}

	chatID, err := s.chatsClient.CreateServerChat(ctx, serverID.String(), name, createdBy.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create chat in chats service: %w", err)
	}

	parsedChatID, err := uuid.Parse(chatID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID from chats service: %w", err)
	}

	if err := s.chatRepo.AddChat(ctx, serverID, parsedChatID, name); err != nil {
		return nil, fmt.Errorf("failed to save chat relation: %w", err)
	}

	return &repository.ServerChat{
		ServerID:  serverID,
		ChatID:    parsedChatID,
		Name:      name,
		Position:  0,
		CreatedAt: time.Now(),
	}, nil
}

func (s *ServerService) ListServerChats(ctx context.Context, serverID, requesterID uuid.UUID) ([]*repository.ServerChat, error) {
	if err := s.checkMemberAccess(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	return s.chatRepo.GetByServer(ctx, serverID)
}

func (s *ServerService) DeleteServerChat(ctx context.Context, serverID, chatID, deleterID uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, deleterID); err != nil {
		return err
	}

	canManage, err := s.canManageChannels(ctx, serverID, deleterID)
	if err != nil {
		return err
	}
	if !canManage {
		return domain.ErrPermissionDenied
	}

	return s.chatRepo.RemoveChat(ctx, serverID, chatID)
}
