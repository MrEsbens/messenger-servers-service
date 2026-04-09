package domain

type Permission string

const (
	PermManageServer   Permission = "manage_server"   // Полный контроль над сервером
	PermManageChannels Permission = "manage_channels" // Создание/удаление каналов
	PermManageRoles    Permission = "manage_roles"    // Управление ролями
	PermKickMembers    Permission = "kick_members"    // Выгонять участников
	PermBanMembers     Permission = "ban_members"     // Банить участников
	PermMuteMembers    Permission = "mute_members"    // Мутить участников
	PermSendMessages   Permission = "send_messages"   // Отправлять сообщения
)

var AllPermissions = []Permission{
	PermManageServer,
	PermManageChannels,
	PermManageRoles,
	PermKickMembers,
	PermBanMembers,
	PermMuteMembers,
	PermSendMessages,
}

// IsValidPermission проверяет, что право существует
func IsValidPermission(permission string) bool {
	for _, p := range AllPermissions {
		if string(p) == permission {
			return true
		}
	}
	return false
}

// HasPermission проверяет, есть ли у участника хотя бы одно из указанных прав
func HasPermission(roles []*ServerRole, permissions ...Permission) bool {
	for _, role := range roles {
		for _, perm := range permissions {
			if role.HasPermission(string(perm)) {
				return true
			}
		}
	}
	return false
}

// HasAllPermissions проверяет, есть ли у участника ВСЕ указанные права
func HasAllPermissions(roles []*ServerRole, permissions ...Permission) bool {
	for _, perm := range permissions {
		found := false
		for _, role := range roles {
			if role.HasPermission(string(perm)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func HasManageServerAccess(roles []*ServerRole) bool {
	return HasPermission(roles, PermManageServer)
}

func HasManageChannelsAccess(roles []*ServerRole) bool {
	return HasPermission(roles, PermManageChannels)
}