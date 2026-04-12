package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	repoMocks "github.com/MrEsbens/messenger-servers-service/internal/repository/mocks"
	"github.com/MrEsbens/messenger-servers-service/internal/service"
	clientMocks "github.com/MrEsbens/messenger-servers-service/internal/transport/grpcclient/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(t *testing.T) (
	service.ServerServiceInterface,
	*repoMocks.MockServerRepository,
	*repoMocks.MockConfigRepository,
	*repoMocks.MockMemberRepository,
	*repoMocks.MockRoleRepository,
	*repoMocks.MockChatRepository,
	*repoMocks.MockModerationRepository,
	*clientMocks.MockIdentityClient,
	*clientMocks.MockChatsClient,
	*clientMocks.MockModerationClient,
) {
	serverRepo := repoMocks.NewMockServerRepository(t)
	configRepo := repoMocks.NewMockConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	roleRepo := repoMocks.NewMockRoleRepository(t)
	moderationRepo := repoMocks.NewMockModerationRepository(t)
	chatRepo := repoMocks.NewMockChatRepository(t)
	identityClient := clientMocks.NewMockIdentityClient(t)
	chatsClient := clientMocks.NewMockChatsClient(t)
	moderationClient := clientMocks.NewMockModerationClient(t)

	svc := service.NewServerService(
		serverRepo,
		configRepo,
		memberRepo,
		roleRepo,
		moderationRepo,
		chatRepo,
		identityClient,
		chatsClient,
		moderationClient,
	)

	return svc, serverRepo, configRepo, memberRepo, roleRepo, chatRepo, moderationRepo, identityClient, chatsClient, moderationClient
}

func newTestServer(ownerID uuid.UUID) *domain.Server {
	return &domain.Server{
		ID:        uuid.New(),
		Name:      "Test Server",
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newTestMember(serverID, userID uuid.UUID) *domain.ServerMember {
	return &domain.ServerMember{
		ID:         uuid.New(),
		ServerID:   serverID,
		UserID:     userID,
		JoinedAt:   time.Now(),
		IsMuted:    false,
		IsBanned:   false,
		MutedUntil: nil,
	}
}

func newTestRole(serverID uuid.UUID, name string, permissions []string) *domain.ServerRole {
	return &domain.ServerRole{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        name,
		Color:       "#99AAB5",
		Permissions: permissions,
		Position:    0,
		IsDefault:   false,
		CreatedAt:   time.Now(),
	}
}

func TestServerService_CreateServer_Success(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, identityClient, _, _ := newTestService(t)

	ownerID := uuid.New()

	identityClient.On("UserExists", mock.Anything, ownerID.String()).Return(true, nil)

	serverRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.Server) bool {
		return s.OwnerID == ownerID && s.Name == "Test Server"
	})).Return(nil)

	memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return m.UserID == ownerID
	})).Return(nil)

	result, err := svc.CreateServer(context.Background(), "Test Server", ownerID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ownerID, result.OwnerID)
	assert.Equal(t, "Test Server", result.Name)

	identityClient.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestServerService_CreateServer_IdentityServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc, _, _, _, _, _, _, identityClient, _, _ := newTestService(t)

	ownerID := uuid.New()

	identityClient.On("UserExists", mock.Anything, ownerID.String()).Return(false, nil)

	_, err := svc.CreateServer(context.Background(), "Test Server", ownerID, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner not found")

	identityClient.AssertExpectations(t)
}

func TestServerService_CreateServer_IdentityServiceError(t *testing.T) {
	t.Parallel()

	svc, _, _, _, _, _, _, identityClient, _, _ := newTestService(t)

	ownerID := uuid.New()

	identityClient.On("UserExists", mock.Anything, ownerID.String()).Return(false, assert.AnError)

	_, err := svc.CreateServer(context.Background(), "Test Server", ownerID, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify owner")

	identityClient.AssertExpectations(t)
}

func TestServerService_CreateServer_InvalidName(t *testing.T) {
	t.Parallel()

	svc, _, _, _, _, _, _, identityClient, _, _ := newTestService(t)

	ownerID := uuid.New()

	identityClient.On("UserExists", mock.Anything, ownerID.String()).Return(true, nil)

	_, err := svc.CreateServer(context.Background(), "", ownerID, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server name is required")

	identityClient.AssertExpectations(t)
}

func TestServerService_UpdateServerConfig_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	config := &domain.ServerConfig{
		ServerID:          serverID,
		MaxMembers:        100,
		MaxChannels:       100,
		ModerationEnabled: true,
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	configRepo.On("UpdateServerConfig", mock.Anything, mock.MatchedBy(func(c *domain.ServerConfig) bool {
		return c.MaxMembers == 100 && c.ModerationEnabled
	})).Return(nil)

	err := svc.UpdateServerConfig(context.Background(), serverID, ownerID, config)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	configRepo.AssertExpectations(t)
}

func TestServerService_UpdateServerConfig_AdminWithManageServer(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	adminMember := newTestMember(serverID, adminID)
	adminMember.ID = uuid.New()

	adminRole := newTestRole(serverID, "Server Admin", []string{"manage_server"})

	config := &domain.ServerConfig{
		ServerID:          serverID,
		MaxMembers:        200,
		MaxChannels:       100,
		ModerationEnabled: true,
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, adminID).Return(adminMember, nil)

	roleRepo.On("GetMemberRoles", mock.Anything, adminMember.ID).Return([]*domain.ServerRole{adminRole}, nil)

	configRepo.On("UpdateServerConfig", mock.Anything, mock.MatchedBy(func(c *domain.ServerConfig) bool {
		return c.MaxMembers == 200 && c.MaxChannels == 100
	})).Return(nil)

	err := svc.UpdateServerConfig(context.Background(), serverID, adminID, config)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	configRepo.AssertExpectations(t)
}

func TestServerService_UpdateServerConfig_NoPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	userMember := newTestMember(serverID, userID)
	userMember.ID = uuid.New()

	userRole := newTestRole(serverID, "Member", []string{"send_messages"})

	config := &domain.ServerConfig{
		ServerID:    serverID,
		MaxMembers:  300,
		MaxChannels: 100,
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(userMember, nil)

	roleRepo.On("GetMemberRoles", mock.Anything, userMember.ID).Return([]*domain.ServerRole{userRole}, nil)

	err := svc.UpdateServerConfig(context.Background(), serverID, userID, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	configRepo.AssertNotCalled(t, "UpdateServerConfig")

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
}

func TestServerService_UpdateServerConfig_ServerNotFound(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	serverRepo.On("GetByID", mock.Anything, serverID).Return(nil, repository.ErrNotFound)

	config := &domain.ServerConfig{ServerID: serverID}

	err := svc.UpdateServerConfig(context.Background(), serverID, ownerID, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestServerService_BanMember_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	targetMember := newTestMember(serverID, targetID)

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)
	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return m.IsBanned
	})).Return(nil)

	err := svc.BanMember(context.Background(), serverID, targetID, ownerID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestServerService_BanMember_WithBanPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	targetMember := newTestMember(serverID, targetID)

	modRole := newTestRole(serverID, "Moderator", []string{"ban_members"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return m.IsBanned
	})).Return(nil)

	err := svc.BanMember(context.Background(), serverID, targetID, modID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
}

func TestServerService_BanMember_NoPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	userMember := newTestMember(serverID, userID)
	userMember.ID = uuid.New()

	userRole := newTestRole(serverID, "Member", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(userMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, userMember.ID).Return([]*domain.ServerRole{userRole}, nil)

	err := svc.BanMember(context.Background(), serverID, targetID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	memberRepo.AssertNotCalled(t, "Update")
}

func TestServerService_BanMember_CannotBanOwner(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	ownerMember := newTestMember(serverID, ownerID)

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	err := svc.BanMember(context.Background(), serverID, ownerID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot ban server owner")
}

func TestServerService_CreateServerChat_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, memberRepo, _, chatRepo, _, _, chatsClient, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	chatID := uuid.New().String()
	parsedChatID := uuid.MustParse(chatID)

	server := newTestServer(ownerID)
	server.ID = serverID

	config := &domain.ServerConfig{
		ServerID:    serverID,
		MaxChannels: 500,
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(config, nil)

	chatRepo.On("GetByServer", mock.Anything, serverID).Return([]*repository.ServerChat{}, nil)

	chatsClient.On("CreateServerChat", mock.Anything, serverID.String(), "general", ownerID.String()).Return(chatID, nil)

	chatRepo.On("AddChat", mock.Anything, serverID, parsedChatID, "general").Return(nil)

	result, err := svc.CreateServerChat(context.Background(), serverID, "general", ownerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "general", result.Name)

	serverRepo.AssertExpectations(t)
	configRepo.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
	chatsClient.AssertExpectations(t)
}

func TestServerService_CreateServerChat_NoManageChannelsPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	userMember := newTestMember(serverID, userID)
	userMember.ID = uuid.New()

	userRole := newTestRole(serverID, "Member", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(userMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, userMember.ID).Return([]*domain.ServerRole{userRole}, nil)

	result, err := svc.CreateServerChat(context.Background(), serverID, "test-channel", userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
	assert.Nil(t, result)
}

func TestServerService_CreateServerChat_ChatsServiceDown(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, memberRepo, _, chatRepo, _, _, chatsClient, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	config := &domain.ServerConfig{
		ServerID:    serverID,
		MaxChannels: 500,
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(config, nil)

	chatRepo.On("GetByServer", mock.Anything, serverID).Return([]*repository.ServerChat{}, nil)

	chatsClient.On("CreateServerChat", mock.Anything, serverID.String(), "general", ownerID.String()).Return("", assert.AnError)

	result, err := svc.CreateServerChat(context.Background(), serverID, "general", ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create chat in chats service")
	assert.Nil(t, result)

	chatsClient.AssertExpectations(t)
}

func TestServerService_CreateServerChat_MaxChannelsReached(t *testing.T) {
	t.Parallel()

	svc, serverRepo, configRepo, memberRepo, _, chatRepo, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	config := &domain.ServerConfig{
		ServerID:    serverID,
		MaxChannels: 2,
	}

	existingChats := []*repository.ServerChat{
		{ServerID: serverID, ChatID: uuid.New(), Name: "chat1"},
		{ServerID: serverID, ChatID: uuid.New(), Name: "chat2"},
	}

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(config, nil)

	chatRepo.On("GetByServer", mock.Anything, serverID).Return(existingChats, nil)

	result, err := svc.CreateServerChat(context.Background(), serverID, "new-channel", ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of channels reached")
	assert.Nil(t, result)
}

func TestServerService_AssignRole_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	targetMemberID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	targetMember := newTestMember(serverID, uuid.New())
	targetMember.ID = targetMemberID

	role := newTestRole(serverID, "Helper", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	roleRepo.On("GetByID", mock.Anything, roleID).Return(role, nil)

	roleRepo.On("AssignToMember", mock.Anything, targetMemberID, roleID).Return(nil)

	err := svc.AssignRole(context.Background(), serverID, targetMemberID, roleID, ownerID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
}

func TestServerService_AssignRole_WithManageRolesPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	targetMemberID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	adminMember := newTestMember(serverID, adminID)
	adminMember.ID = uuid.New()

	targetMember := newTestMember(serverID, uuid.New())
	targetMember.ID = targetMemberID

	adminRole := newTestRole(serverID, "Admin", []string{"manage_roles"})
	targetRole := newTestRole(serverID, "Helper", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, adminID).Return(adminMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, adminMember.ID).Return([]*domain.ServerRole{adminRole}, nil)

	roleRepo.On("GetByID", mock.Anything, roleID).Return(targetRole, nil)

	roleRepo.On("AssignToMember", mock.Anything, targetMemberID, roleID).Return(nil)

	err := svc.AssignRole(context.Background(), serverID, targetMemberID, roleID, adminID)

	assert.NoError(t, err)
}

func TestServerService_AssignRole_NoManageRolesPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	userMember := newTestMember(serverID, userID)
	userMember.ID = uuid.New()

	userRole := newTestRole(serverID, "Member", []string{"send_messages"})

	// targetRole := newTestRole(serverID, "Helper", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(userMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, userMember.ID).Return([]*domain.ServerRole{userRole}, nil)

	err := svc.AssignRole(context.Background(), serverID, uuid.New(), roleID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	roleRepo.AssertNotCalled(t, "AssignToMember")
}

func TestServerService_AssignRole_HierarchyViolation(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetMemberID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	modRole := newTestRole(serverID, "Moderator", []string{"mute_members"})
	modRole.Position = 5

	highRole := newTestRole(serverID, "Admin", []string{"ban_members"})
	highRole.Position = 10
	highRole.ID = roleID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	roleRepo.On("GetByID", mock.Anything, roleID).Return(highRole, nil)

	err := svc.AssignRole(context.Background(), serverID, targetMemberID, roleID, modID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	roleRepo.AssertNotCalled(t, "AssignToMember")
}

func TestServerService_ListServerChats_MemberSuccess(t *testing.T) {
	t.Parallel()

	svc, _, _, memberRepo, _, chatRepo, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	member := newTestMember(serverID, ownerID)

	existingChats := []*repository.ServerChat{
		{ServerID: serverID, ChatID: uuid.New(), Name: "general"},
		{ServerID: serverID, ChatID: uuid.New(), Name: "offtopic"},
	}

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(member, nil)

	chatRepo.On("GetByServer", mock.Anything, serverID).Return(existingChats, nil)

	result, err := svc.ListServerChats(context.Background(), serverID, ownerID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "general", result[0].Name)

	memberRepo.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
}

func TestServerService_ListServerChats_NotMember(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(nil, repository.ErrNotFound)

	result, err := svc.ListServerChats(context.Background(), serverID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is not a member")
	assert.Nil(t, result)
}

func TestServerService_CheckMessageModeration_Disabled(t *testing.T) {
	t.Parallel()

	svc, _, configRepo, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	userID := uuid.New()

	serverConfig := &domain.ServerConfig{
		ServerID:          serverID,
		ModerationEnabled: false,
	}

	configRepo.On("GetModerationConfig", mock.Anything, serverID).Return(&domain.ModerationConfig{}, nil)
	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(serverConfig, nil)

	result, err := svc.CheckMessageModeration(context.Background(), serverID, userID, nil, "Hello, world!")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestServerService_CheckMessageModeration_ServiceDown(t *testing.T) {
	t.Parallel()

	serverRepo := repoMocks.NewMockServerRepository(t)
	configRepo := repoMocks.NewMockConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	roleRepo := repoMocks.NewMockRoleRepository(t)
	moderationRepo := repoMocks.NewMockModerationRepository(t)
	chatRepo := repoMocks.NewMockChatRepository(t)
	identityClient := clientMocks.NewMockIdentityClient(t)
	chatsClient := clientMocks.NewMockChatsClient(t)

	svc := service.NewServerService(
		serverRepo,
		configRepo,
		memberRepo,
		roleRepo,
		moderationRepo,
		chatRepo,
		identityClient,
		chatsClient,
		nil,
	)

	serverID := uuid.New()
	userID := uuid.New()

	serverConfig := &domain.ServerConfig{
		ServerID:          serverID,
		ModerationEnabled: true,
	}

	configRepo.On("GetModerationConfig", mock.Anything, serverID).Return(&domain.ModerationConfig{}, nil)
	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(serverConfig, nil)

	_, err := svc.CheckMessageModeration(context.Background(), serverID, userID, nil, "Hello, world!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "moderation service client not initialized")

	configRepo.AssertExpectations(t)
}

func TestServerService_CheckMessageModeration_MessageAllowed(t *testing.T) {
	t.Parallel()

	svc, _, configRepo, _, _, _, moderationRepo, _, _, moderationClient := newTestService(t)

	serverID := uuid.New()
	userID := uuid.New()
	messageID := uuid.New()

	serverConfig := &domain.ServerConfig{
		ServerID:          serverID,
		ModerationEnabled: true,
	}

	modConfig := &domain.ModerationConfig{
		ServerID:              serverID,
		ProfanityFilterAction: domain.ActionDelete,
	}

	modResult := &domain.ModerationResult{
		Allowed:    true,
		Violations: []domain.Violation{},
	}

	configRepo.On("GetModerationConfig", mock.Anything, serverID).Return(modConfig, nil)
	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(serverConfig, nil)
	moderationClient.On("CheckText", mock.Anything, "Hello, world!", modConfig).Return(modResult, nil)

	result, err := svc.CheckMessageModeration(context.Background(), serverID, userID, &messageID, "Hello, world!")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)

	moderationRepo.AssertNotCalled(t, "Create")
}

func TestServerService_CheckMessageModeration_MessageBlocked(t *testing.T) {
	t.Parallel()

	svc, _, configRepo, _, _, _, moderationRepo, _, _, moderationClient := newTestService(t)

	serverID := uuid.New()
	userID := uuid.New()
	messageID := uuid.New()

	serverConfig := &domain.ServerConfig{
		ServerID:          serverID,
		ModerationEnabled: true,
	}

	modConfig := &domain.ModerationConfig{
		ServerID:              serverID,
		ProfanityFilterAction: domain.ActionDelete,
	}

	modResult := &domain.ModerationResult{
		Allowed: false,
		Violations: []domain.Violation{
			{Type: domain.ViolationProfanity, Message: "profanity detected"},
		},
	}

	configRepo.On("GetModerationConfig", mock.Anything, serverID).Return(modConfig, nil)
	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(serverConfig, nil)
	moderationClient.On("CheckText", mock.Anything, "bad word!", modConfig).Return(modResult, nil)
	moderationRepo.On("Create", mock.Anything, mock.MatchedBy(func(v *domain.ModerationViolation) bool {
		return v.ViolationType == domain.ViolationProfanity && v.ActionTaken == domain.ActionDelete
	})).Return(nil)

	result, err := svc.CheckMessageModeration(context.Background(), serverID, userID, &messageID, "bad word!")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Allowed)

	moderationRepo.AssertExpectations(t)
}

func TestServerService_DeleteServer_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	serverRepo.On("SoftDelete", mock.Anything, serverID).Return(nil)

	err := svc.DeleteServer(context.Background(), serverID, ownerID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
}

func TestServerService_DeleteServer_NotOwner(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	err := svc.DeleteServer(context.Background(), serverID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only server owner can perform this action")

	serverRepo.AssertNotCalled(t, "SoftDelete")
}

func TestServerService_DeleteServer_ServerNotFound(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, _, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	serverRepo.On("GetByID", mock.Anything, serverID).Return(nil, repository.ErrNotFound)

	err := svc.DeleteServer(context.Background(), serverID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestServerService_RemoveMember_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	targetMember := newTestMember(serverID, targetID)

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Delete", mock.Anything, targetMember.ID).Return(nil)

	err := svc.RemoveMember(context.Background(), serverID, targetID, ownerID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestServerService_RemoveMember_WithKickPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	targetMember := newTestMember(serverID, targetID)

	modRole := newTestRole(serverID, "Moderator", []string{"kick_members"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Delete", mock.Anything, targetMember.ID).Return(nil)

	err := svc.RemoveMember(context.Background(), serverID, targetID, modID)

	assert.NoError(t, err)
}

func TestServerService_RemoveMember_CannotKickOwner(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	ownerMember := newTestMember(serverID, ownerID)

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	err := svc.RemoveMember(context.Background(), serverID, ownerID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot kick server owner")
}

func TestServerService_RemoveMember_TargetNotFound(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(nil, repository.ErrNotFound)

	err := svc.RemoveMember(context.Background(), serverID, targetID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is not a member")
}

func TestServerService_MuteMember_WithPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	targetMember := newTestMember(serverID, targetID)

	modRole := newTestRole(serverID, "Moderator", []string{"mute_members"})

	duration := 24 * time.Hour

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return m.IsMuted && m.MutedUntil != nil
	})).Return(nil)

	err := svc.MuteMember(context.Background(), serverID, targetID, modID, &duration)

	assert.NoError(t, err)
}

func TestServerService_MuteMember_CannotMuteOwner(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	ownerMember := newTestMember(serverID, ownerID)

	duration := 1 * time.Hour

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(ownerMember, nil)

	err := svc.MuteMember(context.Background(), serverID, ownerID, ownerID, &duration)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot ban server owner")
}

func TestServerService_UnmuteMember_Success(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	targetMember := newTestMember(serverID, targetID)
	targetMember.IsMuted = true

	modRole := newTestRole(serverID, "Moderator", []string{"mute_members"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return !m.IsMuted && m.MutedUntil == nil
	})).Return(nil)

	err := svc.UnmuteMember(context.Background(), serverID, targetID, modID)

	assert.NoError(t, err)
}

func TestServerService_UnbanMember_Success(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	modID := uuid.New()
	targetID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	modMember := newTestMember(serverID, modID)
	modMember.ID = uuid.New()

	targetMember := newTestMember(serverID, targetID)
	targetMember.IsBanned = true

	modRole := newTestRole(serverID, "Moderator", []string{"ban_members"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, modID).Return(modMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, modMember.ID).Return([]*domain.ServerRole{modRole}, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, targetID).Return(targetMember, nil)

	memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.ServerMember) bool {
		return !m.IsBanned
	})).Return(nil)

	err := svc.UnbanMember(context.Background(), serverID, targetID, modID)

	assert.NoError(t, err)
}

func TestServerService_DeleteRole_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	role := newTestRole(serverID, "Temp Role", []string{})
	role.ID = roleID
	role.IsDefault = false

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	roleRepo.On("GetByID", mock.Anything, roleID).Return(role, nil)

	roleRepo.On("Delete", mock.Anything, roleID).Return(nil)

	err := svc.DeleteRole(context.Background(), serverID, roleID, ownerID)

	assert.NoError(t, err)
}

func TestServerService_DeleteRole_CannotDeleteDefault(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	roleID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	defaultRole := newTestRole(serverID, "@everyone", []string{})
	defaultRole.ID = roleID
	defaultRole.IsDefault = true

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	roleRepo.On("GetByID", mock.Anything, roleID).Return(defaultRole, nil)

	err := svc.DeleteRole(context.Background(), serverID, roleID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete default role")

	roleRepo.AssertNotCalled(t, "Delete")
}

func TestServerService_DeleteServerChat_OwnerSuccess(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, _, chatRepo, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	chatID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	chatRepo.On("RemoveChat", mock.Anything, serverID, chatID).Return(nil)

	err := svc.DeleteServerChat(context.Background(), serverID, chatID, ownerID)

	assert.NoError(t, err)

	serverRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
}

func TestServerService_DeleteServerChat_NoManageChannelsPermission(t *testing.T) {
	t.Parallel()

	svc, serverRepo, _, memberRepo, roleRepo, chatRepo, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	chatID := uuid.New()

	server := newTestServer(ownerID)
	server.ID = serverID

	userMember := newTestMember(serverID, userID)
	userMember.ID = uuid.New()

	userRole := newTestRole(serverID, "Member", []string{"send_messages"})

	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(userMember, nil)
	roleRepo.On("GetMemberRoles", mock.Anything, userMember.ID).Return([]*domain.ServerRole{userRole}, nil)

	err := svc.DeleteServerChat(context.Background(), serverID, chatID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	chatRepo.AssertNotCalled(t, "RemoveChat")
}

func TestServerService_GetMember_Success(t *testing.T) {
	t.Parallel()

	svc, _, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	userID := uuid.New()

	member := newTestMember(serverID, userID)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(member, nil)

	result, err := svc.GetMember(context.Background(), serverID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)

	memberRepo.AssertExpectations(t)
}

func TestServerService_GetMember_NotFound(t *testing.T) {
	t.Parallel()

	svc, _, _, memberRepo, _, _, _, _, _, _ := newTestService(t)

	serverID := uuid.New()
	userID := uuid.New()

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, userID).Return(nil, repository.ErrNotFound)

	result, err := svc.GetMember(context.Background(), serverID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is not a member")
	assert.Nil(t, result)
}

func TestServerService_GetMemberRoles_Success(t *testing.T) {
	t.Parallel()

	svc, _, _, _, roleRepo, _, _, _, _, _ := newTestService(t)

	memberID := uuid.New()

	roles := []*domain.ServerRole{
		newTestRole(uuid.New(), "Role1", []string{"send_messages"}),
		newTestRole(uuid.New(), "Role2", []string{"ban_members"}),
	}

	roleRepo.On("GetMemberRoles", mock.Anything, memberID).Return(roles, nil)

	result, err := svc.GetMemberRoles(context.Background(), memberID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Role1", result[0].Name)

	roleRepo.AssertExpectations(t)
}

func TestServerService_AddMember_MaxMembersReached(t *testing.T) {
	t.Parallel()

	svc, _, configRepo, memberRepo, _, _, _, identityClient, _, _ := newTestService(t)

	serverID := uuid.New()
	ownerID := uuid.New()
	newUserID := uuid.New()

	config := &domain.ServerConfig{
		ServerID:   serverID,
		MaxMembers: 2,
	}

	identityClient.On("UserExists", mock.Anything, newUserID.String()).Return(true, nil)

	memberRepo.On("GetByServerAndUser", mock.Anything, serverID, ownerID).Return(&domain.ServerMember{UserID: ownerID}, nil)

	memberRepo.On("Exists", mock.Anything, serverID, newUserID).Return(false, nil)

	configRepo.On("GetServerConfig", mock.Anything, serverID).Return(config, nil)

	memberRepo.On("CountByServer", mock.Anything, serverID).Return(2, nil)

	err := svc.AddMember(context.Background(), serverID, newUserID, ownerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of members reached")

	memberRepo.AssertNotCalled(t, "Create")
}
