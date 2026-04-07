package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcclient"
	"github.com/google/uuid"
)

// ServerServiceInterface — публичный интерфейс сервиса
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
	ListServerChats(ctx context.Context, serverID uuid.UUID) ([]*repository.ServerChat, error)
	DeleteServerChat(ctx context.Context, serverID, chatID, deleterID uuid.UUID) error

	// Moderation
	CheckMessageModeration(ctx context.Context, serverID, userID uuid.UUID, messageID *uuid.UUID, text string) (*domain.ModerationResult, error)
	LogViolation(ctx context.Context, serverID, userID uuid.UUID, messageID *uuid.UUID, messageContent *string, violationType domain.ViolationType, action domain.ModerationAction) error
}

// ServerService — реализация сервиса
type ServerService struct {
	serverRepo     repository.ServerRepository
	configRepo     repository.ConfigRepository
	memberRepo     repository.MemberRepository
	roleRepo       repository.RoleRepository
	moderationRepo repository.ModerationRepository
	chatRepo       repository.ChatRepository

	// Внешние сервисы
	identityClient   grpcclient.IdentityClientInterface
	chatsClient      grpcclient.ChatsClientInterface
	moderationClient grpcclient.ModerationClientInterface
}

// NewServerService создаёт новый сервис
func NewServerService(
	serverRepo repository.ServerRepository,
	configRepo repository.ConfigRepository,
	memberRepo repository.MemberRepository,
	roleRepo repository.RoleRepository,
	moderationRepo repository.ModerationRepository,
	chatRepo repository.ChatRepository,
	identityClient grpcclient.IdentityClientInterface,
	chatsClient grpcclient.ChatsClientInterface,
	moderationClient grpcclient.ModerationClientInterface,
) ServerServiceInterface {
	return &ServerService{
		serverRepo:       serverRepo,
		configRepo:       configRepo,
		memberRepo:       memberRepo,
		roleRepo:         roleRepo,
		moderationRepo:   moderationRepo,
		chatRepo:         chatRepo,
		identityClient:   identityClient,
		chatsClient:      chatsClient,
		moderationClient: moderationClient,
	}
}

// ───────────────────────────────────────────────────────────
// Server CRUD
// ───────────────────────────────────────────────────────────

func (s *ServerService) CreateServer(ctx context.Context, name string, ownerID uuid.UUID, description *string) (*domain.Server, error) {
	// 1. Проверяем существование владельца через Identity Service
	if s.identityClient != nil {
		exists, err := s.identityClient.UserExists(ctx, ownerID.String())
		if err != nil {
			// Fail-open: логируем ошибку, но продолжаем
			fmt.Printf("⚠️ Identity service error: %v\n", err)
		} else if !exists {
			return nil, fmt.Errorf("owner not found: %s", ownerID)
		}
	}

	// 2. Создаём сервер
	server, err := domain.NewServer(name, ownerID, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create server domain object: %w", err)
	}

	// 3. Сохраняем в БД (конфиги создаются триггером/репозиторием)
	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// 4. Добавляем владельца как участника с дефолтной ролью
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
	// 1. Проверяем, что запрашивающий — участник сервера
	if err := s.checkMemberAccess(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// 2. Получаем сервер
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
	// 1. Проверяем, что updater — владелец
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrServerNotFound
		}
		return err
	}

	if server.OwnerID != updaterID {
		return domain.ErrNotServerOwner
	}

	// 2. Обновляем
	server.Update(name, description)
	return s.serverRepo.Update(ctx, server)
}

func (s *ServerService) DeleteServer(ctx context.Context, serverID, deleterID uuid.UUID) error {
	// 1. Проверяем, что deleter — владелец
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

	// 2. Soft delete
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
			continue // Пропускаем удалённые серверы
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
	// 1. Проверяем, что updater — владелец
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server.OwnerID != updaterID {
		return domain.ErrNotServerOwner
	}

	// 2. Валидируем и обновляем
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
	// 1. Проверяем, что updater — владелец
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server.OwnerID != updaterID {
		return domain.ErrNotServerOwner
	}

	// 2. Валидируем и обновляем
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid moderation config: %w", err)
	}

	return s.configRepo.UpdateModerationConfig(ctx, config)
}

// ───────────────────────────────────────────────────────────
// Members
// ───────────────────────────────────────────────────────────

func (s *ServerService) AddMember(ctx context.Context, serverID, userID, addedBy uuid.UUID) error {
	// 1. Проверяем, что addedBy — участник
	if err := s.checkMemberAccess(ctx, serverID, addedBy); err != nil {
		return err
	}

	if s.identityClient != nil {
		exists, err := s.identityClient.UserExists(ctx, userID.String())
		if err != nil {
			fmt.Printf("⚠️ Identity service error for user %s: %v\n", userID, err)
		} else if !exists {
			return fmt.Errorf("user not found: %s", userID)
		}
	}

	// 2. Проверяем, не существует ли уже участник
	exists, err := s.memberRepo.Exists(ctx, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to check member exists: %w", err)
	}
	if exists {
		return domain.ErrAlreadyMember
	}

	// 3. Проверяем лимит участников
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

	// 4. Создаём участника
	member, err := domain.NewServerMember(serverID, userID)
	if err != nil {
		return err
	}
	return s.memberRepo.Create(ctx, member)
}

func (s *ServerService) RemoveMember(ctx context.Context, serverID, userID, removedBy uuid.UUID) error {
	// 1. Проверяем, что removedBy — участник
	if err := s.checkMemberAccess(ctx, serverID, removedBy); err != nil {
		return err
	}

	// 2. Получаем участника
	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	// 3. Нельзя удалить владельца
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if member.UserID == server.OwnerID {
		return domain.ErrCannotKickOwner
	}

	// 4. Удаляем
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
	// 1. Проверяем права banning пользователя
	if err := s.checkMemberAccess(ctx, serverID, bannedBy); err != nil {
		return err
	}

	// 2. Получаем участника
	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	// 3. Нельзя забанить владельца
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if member.UserID == server.OwnerID {
		return domain.ErrCannotBanOwner
	}

	// 4. Баняем
	member.Ban()
	return s.memberRepo.Update(ctx, member)
}

func (s *ServerService) UnbanMember(ctx context.Context, serverID, userID, unbannedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, unbannedBy); err != nil {
		return err
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

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	member.Mute(duration)
	return s.memberRepo.Update(ctx, member)
}

func (s *ServerService) UnmuteMember(ctx context.Context, serverID, userID, unmutedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, unmutedBy); err != nil {
		return err
	}

	member, err := s.memberRepo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return domain.ErrNotMember
		}
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err == nil && member.UserID == server.OwnerID {
		return domain.ErrCannotMuteOwner
	}

	// 4. Размучиваем
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

	return s.roleRepo.AssignToMember(ctx, memberID, roleID)
}

func (s *ServerService) RemoveRole(ctx context.Context, serverID, memberID, roleID uuid.UUID, removedBy uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, removedBy); err != nil {
		return err
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
	// 1. Проверяем доступ
	if err := s.checkMemberAccess(ctx, serverID, createdBy); err != nil {
		return nil, err
	}

	// 2. Проверяем лимит каналов
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

	// 3. Создаём чат через Chats Service
	var chatID uuid.UUID
	if s.chatsClient != nil {
		id, err := s.chatsClient.CreateServerChat(ctx, serverID.String(), name, createdBy.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create chat in chats service: %w", err)
		}
		chatID, _ = uuid.Parse(id)
	} else {
		// Fallback: генерируем ID локально (для тестов)
		chatID = uuid.New()
	}

	// 4. Сохраняем связь
	if err := s.chatRepo.AddChat(ctx, serverID, chatID, name); err != nil {
		return nil, err
	}

	return &repository.ServerChat{
		ServerID:  serverID,
		ChatID:    chatID,
		Name:      name,
		Position:  0,
		CreatedAt: time.Now(),
	}, nil
}

func (s *ServerService) ListServerChats(ctx context.Context, serverID uuid.UUID) ([]*repository.ServerChat, error) {
	err := s.checkMemberAccess(ctx, serverID, uuid.Nil) // Любой участник может видеть чаты
	if err != nil {
		return nil, err
	}

	return s.chatRepo.GetByServer(ctx, serverID)
}

func (s *ServerService) DeleteServerChat(ctx context.Context, serverID, chatID, deleterID uuid.UUID) error {
	if err := s.checkMemberAccess(ctx, serverID, deleterID); err != nil {
		return err
	}

	return s.chatRepo.RemoveChat(ctx, serverID, chatID)
}

// ───────────────────────────────────────────────────────────
// Moderation
// ───────────────────────────────────────────────────────────

func (s *ServerService) CheckMessageModeration(ctx context.Context, serverID, userID uuid.UUID, messageID *uuid.UUID, text string) (*domain.ModerationResult, error) {
	// 1. Получаем конфиг модерации
	config, err := s.configRepo.GetModerationConfig(ctx, serverID)
	if err != nil {
		// Если конфига нет — пропускаем модерацию
		return &domain.ModerationResult{Allowed: true}, nil
	}

	// 2. Проверяем, включена ли модерация
	serverConfig, err := s.configRepo.GetServerConfig(ctx, serverID)
	if err != nil {
		return &domain.ModerationResult{Allowed: true}, nil
	}
	if !serverConfig.ModerationEnabled {
		return &domain.ModerationResult{Allowed: true}, nil
	}

	// 3. Отправляем в Moderation Service
	if s.moderationClient == nil {
		// Нет клиента — пропускаем
		return &domain.ModerationResult{Allowed: true}, nil
	}

	result, err := s.moderationClient.CheckText(ctx, text, config)
	if err != nil {
		// Fail-open: ошибка ML — пропускаем сообщение
		fmt.Printf("⚠️ Moderation service error: %v\n", err)
		return &domain.ModerationResult{Allowed: true, Fallback: true}, nil
	}

	// 4. Если есть нарушения — логируем и выполняем действия
	if !result.Allowed {
		for _, violation := range result.Violations {
			action := config.GetActionForFilter(string(violation.Type))
			if action != domain.ActionNone {
				_ = s.LogViolation(ctx, serverID, userID, messageID, &text, violation.Type, action)
			}
		}
	}

	return result, nil
}

func (s *ServerService) LogViolation(ctx context.Context, serverID, userID uuid.UUID, messageID *uuid.UUID, messageContent *string, violationType domain.ViolationType, action domain.ModerationAction) error {
	violation := domain.NewModerationViolation(serverID, userID, messageID, messageContent, violationType, action)
	return s.moderationRepo.Create(ctx, violation)
}

// ───────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────

// checkMemberAccess проверяет, что пользователь — участник сервера
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
