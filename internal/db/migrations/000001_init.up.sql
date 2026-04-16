SET SEARCH_PATH TO servers;

CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_servers_owner ON servers(owner_id);
CREATE INDEX idx_servers_deleted ON servers(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS server_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    
    max_members INT DEFAULT 500,
    max_channels INT DEFAULT 500,
    
    default_notification_mode VARCHAR(20) DEFAULT 'all',
    
    moderation_enabled BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_server_configs_server ON server_configs(server_id);

CREATE TABLE IF NOT EXISTS moderation_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    
    -- ML-фильтры: none | warn | mute | ban | delete
    profanity_filter_action VARCHAR(20) DEFAULT 'none',    -- Мат/оскорбления
    toxicity_filter_action VARCHAR(20) DEFAULT 'none',     -- Токсичность (ML)
    nsfw_text_filter_action VARCHAR(20) DEFAULT 'none',    -- 18+ текст (ML)
    hate_speech_filter_action VARCHAR(20) DEFAULT 'none',  -- Хейт-спич (ML)
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_moderation_configs_server ON moderation_configs(server_id);

CREATE TABLE IF NOT EXISTS moderation_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    message_id UUID,
    message_content TEXT,
    
    violation_type VARCHAR(50) NOT NULL,
    action_taken VARCHAR(20),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_violations_server ON moderation_violations(server_id);
CREATE INDEX idx_violations_user ON moderation_violations(user_id);
CREATE INDEX idx_violations_created ON moderation_violations(created_at);

CREATE TABLE IF NOT EXISTS server_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    is_muted BOOLEAN DEFAULT FALSE,
    is_banned BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    UNIQUE(server_id, user_id)
);

CREATE INDEX idx_server_members_user ON server_members(user_id);
CREATE INDEX idx_server_members_server ON server_members(server_id);

CREATE TABLE IF NOT EXISTS server_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(7) DEFAULT '#99AAB5',
    permissions TEXT[] DEFAULT ARRAY[]::TEXT[],
    position INT DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(server_id, name)
);

CREATE INDEX idx_server_roles_server ON server_roles(server_id);

CREATE TABLE IF NOT EXISTS member_roles (
    member_id UUID NOT NULL REFERENCES server_members(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES server_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (member_id, role_id)
);

CREATE TABLE IF NOT EXISTS server_chats (
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    chat_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    position INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (server_id, chat_id)
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_server_configs_updated_at BEFORE UPDATE ON server_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_moderation_configs_updated_at BEFORE UPDATE ON moderation_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();