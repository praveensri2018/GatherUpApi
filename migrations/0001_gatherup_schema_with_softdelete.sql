-- migrations/0001_gatherup_schema_with_softdelete.sql
USE GatherUpDB;
GO
SET NOCOUNT ON;
GO

-- ======================================================================
-- Notes:
-- - Minimal, production-ready schema for a social/chat/tournament app.
-- - Soft-delete built-in: columns is_deleted BIT DEFAULT 0, deleted_at DATETIMEOFFSET NULL.
-- - Avoided multiple-cascade-paths by using ON DELETE NO ACTION / SET NULL where needed.
-- - Add spatial indexes separately after populating geography columns.
-- ======================================================================

/* ---------- helper: create soft-delete columns inline ---------- */
-- We'll define tables with is_deleted and deleted_at columns baked in.

-- ======================================================================
-- 1) Lookup tables
-- ======================================================================
IF OBJECT_ID('dbo.visibility_types','U') IS NULL
BEGIN
  CREATE TABLE dbo.visibility_types (
    id TINYINT PRIMARY KEY,
    code NVARCHAR(50) NOT NULL UNIQUE,
    description NVARCHAR(255) NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
  INSERT INTO dbo.visibility_types (id, code, description) VALUES
    (0,'private','Visible to selected users only'),
    (1,'contacts','Visible to contacts/friends'),
    (2,'public','Visible to everyone'),
    (3,'group','Visible to group members');
END
GO

IF OBJECT_ID('dbo.reaction_types','U') IS NULL
BEGIN
  CREATE TABLE dbo.reaction_types (
    id TINYINT PRIMARY KEY,
    code NVARCHAR(50) NOT NULL UNIQUE,
    icon NVARCHAR(100) NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
  INSERT INTO dbo.reaction_types (id, code, icon) VALUES
    (1,'like','thumbs-up'), (2,'love','heart'), (3,'laugh','laugh'),
    (4,'wow','surprise'), (5,'sad','sad'), (6,'angry','angry');
END
GO

IF OBJECT_ID('dbo.game_types','U') IS NULL
BEGIN
  CREATE TABLE dbo.game_types (
    id TINYINT PRIMARY KEY,
    code NVARCHAR(50) NOT NULL UNIQUE,
    name NVARCHAR(100) NOT NULL,
    description NVARCHAR(255) NULL,
    min_players INT DEFAULT 1,
    max_players INT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
  INSERT INTO dbo.game_types (id, code, name, min_players, max_players) VALUES
    (1,'cricket','Cricket',2,22),(2,'chess','Chess',2,2),(3,'football','Football',2,22);
END
GO

IF OBJECT_ID('dbo.post_categories','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_categories (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(100) NOT NULL UNIQUE,
    description NVARCHAR(255) NULL,
    icon NVARCHAR(100) NULL,
    is_active BIT DEFAULT 1,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
  INSERT INTO dbo.post_categories (name, description) VALUES
    ('sports','Sports related posts'), ('events','Local events'), ('general','General');
END
GO

-- ======================================================================
-- 2) Core users & auth
-- ======================================================================
IF OBJECT_ID('dbo.users','U') IS NULL
BEGIN
  CREATE TABLE dbo.users (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    mobile_number NVARCHAR(32) NOT NULL,
    mobile_normalized NVARCHAR(24) NOT NULL,
    country_code NVARCHAR(8) NULL,
    display_name NVARCHAR(200) NULL,
    avatar_url NVARCHAR(2048) NULL,
    bio NVARCHAR(1000) NULL,
    email NVARCHAR(320) NULL,
    username NVARCHAR(100) NULL,

    latitude DECIMAL(9,6) NULL,
    longitude DECIMAL(9,6) NULL,
    location GEOGRAPHY NULL,
    location_updated_at DATETIMEOFFSET NULL,

    date_of_birth DATE NULL,
    gender NVARCHAR(20) NULL,

    is_mobile_verified BIT NOT NULL DEFAULT 0,
    mobile_verified_at DATETIMEOFFSET NULL,
    is_email_verified BIT NULL DEFAULT 0,
    is_active BIT NOT NULL DEFAULT 1,

    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    updated_at DATETIMEOFFSET NULL,

    -- soft-delete
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,

    rv ROWVERSION NOT NULL
  );

  CREATE UNIQUE INDEX ux_users_mobile_number ON dbo.users(mobile_number);
  CREATE UNIQUE INDEX ux_users_mobile_normalized ON dbo.users(mobile_normalized);
  CREATE UNIQUE INDEX ux_users_username ON dbo.users(username) WHERE username IS NOT NULL AND is_deleted = 0;
  CREATE INDEX idx_users_country ON dbo.users(country_code);
  CREATE INDEX idx_users_created_at ON dbo.users(created_at);
END
GO

IF OBJECT_ID('dbo.user_credentials','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_credentials (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    credential_type NVARCHAR(50) NOT NULL,
    credential_identifier NVARCHAR(512) NULL,
    password_hash NVARCHAR(1024) NULL,
    salt VARBINARY(64) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    last_used DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_usercred_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_usercred_user ON dbo.user_credentials(user_id);
  CREATE INDEX idx_usercred_type ON dbo.user_credentials(credential_type);
  CREATE UNIQUE INDEX ux_usercred_identifier ON dbo.user_credentials(credential_type, credential_identifier) WHERE credential_identifier IS NOT NULL AND is_deleted = 0;
END
GO

IF OBJECT_ID('dbo.user_sessions','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_sessions (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    device_info NVARCHAR(512) NULL,
    ip_address NVARCHAR(45) NULL,
    session_latitude DECIMAL(9,6) NULL,
    session_longitude DECIMAL(9,6) NULL,
    session_location GEOGRAPHY NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    expires_at DATETIMEOFFSET NULL,
    last_activity_at DATETIMEOFFSET NULL,
    is_revoked BIT DEFAULT 0,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_usersessions_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_sessions_user ON dbo.user_sessions(user_id);
  CREATE INDEX idx_sessions_expires ON dbo.user_sessions(expires_at) WHERE is_revoked = 0;
END
GO

IF OBJECT_ID('dbo.user_preferences','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_preferences (
    user_id UNIQUEIDENTIFIER PRIMARY KEY,
    notify_on_message BIT DEFAULT 1,
    notify_on_like BIT DEFAULT 1,
    notify_on_comment BIT DEFAULT 1,
    timezone NVARCHAR(50) NULL,
    lang NVARCHAR(10) NULL,
    theme NVARCHAR(20) DEFAULT 'light',
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_userprefs_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
  );
END
GO

IF OBJECT_ID('dbo.user_skills','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_skills (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    game_type_id TINYINT NOT NULL,
    skill_level NVARCHAR(50) NULL,
    experience_years INT NULL,
    is_public BIT DEFAULT 1,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_userskills_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_userskills_gametype FOREIGN KEY (game_type_id) REFERENCES dbo.game_types(id),
    CONSTRAINT ux_user_skill UNIQUE (user_id, game_type_id)
  );
  CREATE INDEX idx_user_skills_gametype ON dbo.user_skills(game_type_id);
END
GO

-- ======================================================================
-- 3) Social & relationships
-- ======================================================================
IF OBJECT_ID('dbo.contacts','U') IS NULL
BEGIN
  CREATE TABLE dbo.contacts (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    contact_user_id UNIQUEIDENTIFIER NOT NULL,
    relation_type NVARCHAR(50) NOT NULL DEFAULT 'friend',
    status NVARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    accepted_at DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_contacts_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_contacts_contactuser FOREIGN KEY (contact_user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_contacts_user_contact UNIQUE (user_id, contact_user_id)
  );
  CREATE INDEX idx_contacts_user ON dbo.contacts(user_id);
  CREATE INDEX idx_contacts_status ON dbo.contacts(status);
END
GO

IF OBJECT_ID('dbo.blocks','U') IS NULL
BEGIN
  CREATE TABLE dbo.blocks (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    blocked_user_id UNIQUEIDENTIFIER NOT NULL,
    reason NVARCHAR(500) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_blocks_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_blocks_blocked FOREIGN KEY (blocked_user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_block_pair UNIQUE (user_id, blocked_user_id)
  );
  CREATE INDEX idx_blocks_user ON dbo.blocks(user_id);
END
GO

-- ======================================================================
-- 4) Posts system (feed)
-- ======================================================================
IF OBJECT_ID('dbo.posts','U') IS NULL
BEGIN
  CREATE TABLE dbo.posts (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    author_id UNIQUEIDENTIFIER NOT NULL,
    title NVARCHAR(255) NULL,
    body NVARCHAR(MAX) NULL,
    kind NVARCHAR(50) NOT NULL DEFAULT 'text',
    latitude DECIMAL(9,6) NULL,
    longitude DECIMAL(9,6) NULL,
    location GEOGRAPHY NULL,
    location_accuracy INT NULL,
    category_id INT NULL,
    visibility_id TINYINT NOT NULL DEFAULT 2,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    updated_at DATETIMEOFFSET NULL,
    rv ROWVERSION NOT NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_posts_author FOREIGN KEY (author_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_posts_visibility FOREIGN KEY (visibility_id) REFERENCES dbo.visibility_types(id),
    CONSTRAINT fk_posts_category FOREIGN KEY (category_id) REFERENCES dbo.post_categories(id)
  );
  CREATE INDEX idx_posts_author_created ON dbo.posts(author_id, created_at DESC);
  CREATE INDEX idx_posts_visibility ON dbo.posts(visibility_id, created_at DESC);
  CREATE INDEX idx_posts_category ON dbo.posts(category_id, created_at DESC);
  CREATE INDEX idx_posts_kind_created_at ON dbo.posts(kind, created_at DESC);
END
GO

IF OBJECT_ID('dbo.post_media','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_media (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    media_url NVARCHAR(2048) NOT NULL,
    media_type NVARCHAR(50) NULL,
    thumbnail_url NVARCHAR(2048) NULL,
    file_size BIGINT NULL,
    duration_seconds INT NULL,
    sort_order INT DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postmedia_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_post_media_post ON dbo.post_media(post_id);
END
GO

IF OBJECT_ID('dbo.post_counters','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_counters (
    post_id UNIQUEIDENTIFIER PRIMARY KEY,
    view_count BIGINT DEFAULT 0,
    reaction_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    share_count BIGINT DEFAULT 0,
    last_updated DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postcounters_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_post_counters_views ON dbo.post_counters(view_count DESC);
END
GO

IF OBJECT_ID('dbo.post_recipients','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_recipients (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    recipient_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postrecip_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_postrecip_user FOREIGN KEY (recipient_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_post_recipient UNIQUE(post_id, recipient_id)
  );
  CREATE INDEX idx_post_recip_on_recipient ON dbo.post_recipients(recipient_id);
END
GO

IF OBJECT_ID('dbo.post_views','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_views (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    viewer_id UNIQUEIDENTIFIER NULL,
    viewer_ip NVARCHAR(45) NULL,
    viewer_latitude DECIMAL(9,6) NULL,
    viewer_longitude DECIMAL(9,6) NULL,
    viewer_location GEOGRAPHY NULL,
    viewed_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postviews_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_postviews_viewer FOREIGN KEY (viewer_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_post_views_post ON dbo.post_views(post_id);
END
GO

IF OBJECT_ID('dbo.post_reactions','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_reactions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    reaction_type_id TINYINT NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postreact_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_postreact_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_postreact_type FOREIGN KEY (reaction_type_id) REFERENCES dbo.reaction_types(id)
  );
  CREATE UNIQUE INDEX ux_post_reaction_per_user ON dbo.post_reactions(post_id, user_id);
  CREATE INDEX idx_post_reactions_post ON dbo.post_reactions(post_id);
END
GO

IF OBJECT_ID('dbo.post_shares','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_shares (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    sharer_id UNIQUEIDENTIFIER NOT NULL,
    share_comment NVARCHAR(1000) NULL,
    shared_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postshares_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_postshares_sharer FOREIGN KEY (sharer_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_post_shares_post ON dbo.post_shares(post_id);
END
GO

IF OBJECT_ID('dbo.hashtags','U') IS NULL
BEGIN
  CREATE TABLE dbo.hashtags (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tag NVARCHAR(200) NOT NULL UNIQUE,
    usage_count BIGINT DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
END
GO

IF OBJECT_ID('dbo.post_hashtags','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_hashtags (
    post_id UNIQUEIDENTIFIER NOT NULL,
    hashtag_id BIGINT NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT pk_post_hashtags PRIMARY KEY (post_id, hashtag_id),
    CONSTRAINT fk_posthashtags_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_posthashtags_tag FOREIGN KEY (hashtag_id) REFERENCES dbo.hashtags(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_post_hashtags_tag ON dbo.post_hashtags(hashtag_id);
END
GO

IF OBJECT_ID('dbo.post_mentions','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_mentions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    mentioned_user_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_postmentions_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_postmentions_user FOREIGN KEY (mentioned_user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_post_mention UNIQUE (post_id, mentioned_user_id)
  );
  CREATE INDEX idx_post_mentions_user ON dbo.post_mentions(mentioned_user_id);
END
GO

-- ======================================================================
-- 5) Comments & comment reactions
-- ======================================================================
IF OBJECT_ID('dbo.comments','U') IS NULL
BEGIN
  CREATE TABLE dbo.comments (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL,
    author_id UNIQUEIDENTIFIER NOT NULL,
    parent_comment_id BIGINT NULL,
    body NVARCHAR(MAX) NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    updated_at DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_comments_post FOREIGN KEY (post_id) REFERENCES dbo.posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_comments_parent FOREIGN KEY (parent_comment_id) REFERENCES dbo.comments(id) ON DELETE NO ACTION,
    CONSTRAINT fk_comments_author FOREIGN KEY (author_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_comments_post ON dbo.comments(post_id, created_at DESC);
END
GO

IF OBJECT_ID('dbo.comment_reactions','U') IS NULL
BEGIN
  CREATE TABLE dbo.comment_reactions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    comment_id BIGINT NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    reaction_type_id TINYINT NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_commentreact_comment FOREIGN KEY (comment_id) REFERENCES dbo.comments(id) ON DELETE CASCADE,
    CONSTRAINT fk_commentreact_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_commentreact_type FOREIGN KEY (reaction_type_id) REFERENCES dbo.reaction_types(id)
  );
  CREATE UNIQUE INDEX ux_comment_reaction ON dbo.comment_reactions(comment_id, user_id);
END
GO

-- ======================================================================
-- 6) Chat & Messaging
-- ======================================================================
IF OBJECT_ID('dbo.chats','U') IS NULL
BEGIN
  CREATE TABLE dbo.chats (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    uuid AS CONVERT(NVARCHAR(36), id) PERSISTED,
    title NVARCHAR(255) NULL,
    description NVARCHAR(1000) NULL,
    is_group BIT NOT NULL DEFAULT 0,
    group_avatar_url NVARCHAR(2048) NULL,
    created_by UNIQUEIDENTIFIER NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    updated_at DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_chats_created_by FOREIGN KEY (created_by) REFERENCES dbo.users(id) ON DELETE SET NULL
  );
  CREATE INDEX idx_chats_created_at ON dbo.chats(created_at DESC);
END
GO

IF OBJECT_ID('dbo.chat_participants','U') IS NULL
BEGIN
  CREATE TABLE dbo.chat_participants (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    chat_id UNIQUEIDENTIFIER NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    joined_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    left_at DATETIMEOFFSET NULL,
    is_admin BIT DEFAULT 0,
    nickname NVARCHAR(100) NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_chatpart_chat FOREIGN KEY (chat_id) REFERENCES dbo.chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_chatpart_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_chat_participant UNIQUE (chat_id, user_id)
  );
  CREATE INDEX idx_chat_part_userid ON dbo.chat_participants(user_id);
END
GO

IF OBJECT_ID('dbo.muted_chats','U') IS NULL
BEGIN
  CREATE TABLE dbo.muted_chats (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    chat_id UNIQUEIDENTIFIER NOT NULL,
    muted_until DATETIMEOFFSET NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT ux_muted_chat UNIQUE (user_id, chat_id),
    CONSTRAINT fk_muted_chats_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_muted_chats_chat FOREIGN KEY (chat_id) REFERENCES dbo.chats(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_muted_chat_user ON dbo.muted_chats(user_id);
END
GO

IF OBJECT_ID('dbo.messages','U') IS NULL
BEGIN
  CREATE TABLE dbo.messages (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    chat_id UNIQUEIDENTIFIER NOT NULL,
    sender_id UNIQUEIDENTIFIER NOT NULL,
    body NVARCHAR(MAX) NULL,
    kind NVARCHAR(50) DEFAULT 'text',
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    edited_at DATETIMEOFFSET NULL,
    rv ROWVERSION NOT NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_messages_chat FOREIGN KEY (chat_id) REFERENCES dbo.chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_sender FOREIGN KEY (sender_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_messages_chat_created ON dbo.messages(chat_id, created_at DESC);
  CREATE INDEX idx_messages_sender ON dbo.messages(sender_id);
END
GO

IF OBJECT_ID('dbo.message_media','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_media (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL,
    media_url NVARCHAR(2048) NOT NULL,
    media_type NVARCHAR(50) NULL,
    thumbnail_url NVARCHAR(2048) NULL,
    file_size BIGINT NULL,
    duration_seconds INT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_message_media_message FOREIGN KEY (message_id) REFERENCES dbo.messages(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_message_media_msg ON dbo.message_media(message_id);
END
GO

IF OBJECT_ID('dbo.message_reactions','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_reactions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    reaction_type_id TINYINT NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_msg_react_message FOREIGN KEY (message_id) REFERENCES dbo.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_msg_react_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_msg_react_type FOREIGN KEY (reaction_type_id) REFERENCES dbo.reaction_types(id)
  );
  CREATE UNIQUE INDEX ux_msg_reaction ON dbo.message_reactions(message_id, user_id, reaction_type_id);
END
GO

IF OBJECT_ID('dbo.message_status','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_status (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL,
    recipient_id UNIQUEIDENTIFIER NOT NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'sent',
    status_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_msg_status_message FOREIGN KEY (message_id) REFERENCES dbo.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_msg_status_recipient FOREIGN KEY (recipient_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_msg_status_unique UNIQUE (message_id, recipient_id)
  );
  CREATE INDEX idx_msg_status_message ON dbo.message_status(message_id);
  CREATE INDEX idx_msg_status_recipient ON dbo.message_status(recipient_id);
END
GO

IF OBJECT_ID('dbo.message_mentions','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_mentions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL,
    mentioned_user_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_msgmentions_message FOREIGN KEY (message_id) REFERENCES dbo.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_msgmentions_user FOREIGN KEY (mentioned_user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_message_mention UNIQUE (message_id, mentioned_user_id)
  );
  CREATE INDEX idx_msg_mentions_user ON dbo.message_mentions(mentioned_user_id);
END
GO

IF OBJECT_ID('dbo.chat_read_cursors','U') IS NULL
BEGIN
  CREATE TABLE dbo.chat_read_cursors (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    chat_id UNIQUEIDENTIFIER NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    last_read_message_id UNIQUEIDENTIFIER NULL,
    last_read_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_chatread_chat FOREIGN KEY (chat_id) REFERENCES dbo.chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_chatread_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_chatread_lastmsg FOREIGN KEY (last_read_message_id) REFERENCES dbo.messages(id) ON DELETE NO ACTION,
    CONSTRAINT ux_chat_read UNIQUE (chat_id, user_id)
  );
  CREATE INDEX idx_chat_read_chat_user ON dbo.chat_read_cursors(chat_id, user_id);
END
GO

IF OBJECT_ID('dbo.chat_summaries','U') IS NULL
BEGIN
  CREATE TABLE dbo.chat_summaries (
    chat_id UNIQUEIDENTIFIER PRIMARY KEY,
    last_message_id UNIQUEIDENTIFIER NULL,
    last_message_body NVARCHAR(4000) NULL,
    last_message_at DATETIMEOFFSET NULL,
    last_sender_id UNIQUEIDENTIFIER NULL,
    unread_count INT NOT NULL DEFAULT 0,
    participant_count INT NOT NULL DEFAULT 0,
    updated_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_chatsumm_chat FOREIGN KEY (chat_id) REFERENCES dbo.chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_chatsumm_lastmsg FOREIGN KEY (last_message_id) REFERENCES dbo.messages(id) ON DELETE NO ACTION,
    CONSTRAINT fk_chatsumm_lastsender FOREIGN KEY (last_sender_id) REFERENCES dbo.users(id) ON DELETE SET NULL
  );
  CREATE INDEX idx_chat_summary_updated ON dbo.chat_summaries(updated_at DESC);
END
GO

-- ======================================================================
-- 7) Tournaments & games
-- ======================================================================
IF OBJECT_ID('dbo.tournaments','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournaments (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    title NVARCHAR(255) NOT NULL,
    description NVARCHAR(1000) NULL,
    game_type_id TINYINT NOT NULL,
    creator_id UNIQUEIDENTIFIER NOT NULL,
    visibility_id TINYINT NOT NULL DEFAULT 2,
    max_players INT NOT NULL,
    current_players INT NOT NULL DEFAULT 1,
    status NVARCHAR(50) NOT NULL DEFAULT 'pending',
    venue_name NVARCHAR(255) NULL,
    venue_address NVARCHAR(500) NULL,
    latitude DECIMAL(9,6) NULL,
    longitude DECIMAL(9,6) NULL,
    location GEOGRAPHY NULL,
    start_time DATETIMEOFFSET NOT NULL,
    end_time DATETIMEOFFSET NULL,
    registration_deadline DATETIMEOFFSET NULL,
    game_settings NVARCHAR(MAX) NULL,
    rules_text NVARCHAR(MAX) NULL,
    entry_fee DECIMAL(10,2) NULL,
    prize_pool NVARCHAR(500) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_tournaments_game_type FOREIGN KEY (game_type_id) REFERENCES dbo.game_types(id),
    CONSTRAINT fk_tournaments_creator FOREIGN KEY (creator_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_tournaments_visibility FOREIGN KEY (visibility_id) REFERENCES dbo.visibility_types(id)
  );
  CREATE INDEX idx_tournaments_creator ON dbo.tournaments(creator_id);
  CREATE INDEX idx_tournaments_game_type ON dbo.tournaments(game_type_id);
END
GO

IF OBJECT_ID('dbo.tournament_participants','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_participants (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tournament_id UNIQUEIDENTIFIER NOT NULL,
    user_id UNIQUEIDENTIFIER NOT NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'invited',
    team_name NVARCHAR(255) NULL,
    player_role NVARCHAR(100) NULL,
    joined_at DATETIMEOFFSET NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_tournamentpart_tournament FOREIGN KEY (tournament_id) REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk_tournamentpart_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT ux_tournament_participant UNIQUE (tournament_id, user_id)
  );
  CREATE INDEX idx_tournament_part_user ON dbo.tournament_participants(user_id);
END
GO

IF OBJECT_ID('dbo.tournament_matches','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_matches (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    tournament_id UNIQUEIDENTIFIER NOT NULL,
    match_number INT NOT NULL,
    round_number INT NOT NULL DEFAULT 1,
    match_name NVARCHAR(255) NULL,
    participant1_id BIGINT NULL,
    participant2_id BIGINT NULL,
    match_date DATETIMEOFFSET NULL,
    venue NVARCHAR(255) NULL,
    match_latitude DECIMAL(9,6) NULL,
    match_longitude DECIMAL(9,6) NULL,
    match_location GEOGRAPHY NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'scheduled',
    winner_id BIGINT NULL,
    score_participant1 NVARCHAR(200) NULL,
    score_participant2 NVARCHAR(200) NULL,
    match_result NVARCHAR(MAX) NULL,
    match_summary NVARCHAR(1000) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_tournamentmatches_tournament FOREIGN KEY (tournament_id) REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk_tournamentmatches_part1 FOREIGN KEY (participant1_id) REFERENCES dbo.tournament_participants(id) ON DELETE NO ACTION,
    CONSTRAINT fk_tournamentmatches_part2 FOREIGN KEY (participant2_id) REFERENCES dbo.tournament_participants(id) ON DELETE NO ACTION,
    CONSTRAINT fk_tournamentmatches_winner FOREIGN KEY (winner_id) REFERENCES dbo.tournament_participants(id) ON DELETE NO ACTION
  );
  CREATE UNIQUE INDEX ux_tournament_match_number ON dbo.tournament_matches(tournament_id, match_number);
END
GO

IF OBJECT_ID('dbo.tournament_leaderboard','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_leaderboard (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tournament_id UNIQUEIDENTIFIER NOT NULL,
    participant_id BIGINT NOT NULL,
    matches_played INT DEFAULT 0,
    matches_won INT DEFAULT 0,
    matches_lost INT DEFAULT 0,
    points INT DEFAULT 0,
    rank INT NULL,
    additional_stats NVARCHAR(MAX) NULL,
    last_updated DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_leaderboard_tournament FOREIGN KEY (tournament_id) REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk_leaderboard_participant FOREIGN KEY (participant_id) REFERENCES dbo.tournament_participants(id) ON DELETE NO ACTION,
    CONSTRAINT ux_leaderboard_entry UNIQUE (tournament_id, participant_id)
  );
  CREATE INDEX idx_leaderboard_points ON dbo.tournament_leaderboard(tournament_id, points DESC);
END
GO

IF OBJECT_ID('dbo.tournament_shares','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_shares (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tournament_id UNIQUEIDENTIFIER NOT NULL,
    sharer_id UNIQUEIDENTIFIER NOT NULL,
    share_comment NVARCHAR(1000) NULL,
    shared_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_tournamentshares_tournament FOREIGN KEY (tournament_id) REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk_tournamentshares_sharer FOREIGN KEY (sharer_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_tournament_shares_tournament ON dbo.tournament_shares(tournament_id);
END
GO

-- ======================================================================
-- 8) Notifications & system tables
-- ======================================================================
IF OBJECT_ID('dbo.notifications','U') IS NULL
BEGIN
  CREATE TABLE dbo.notifications (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    actor_id UNIQUEIDENTIFIER NULL,
    kind NVARCHAR(100) NOT NULL,
    reference_type NVARCHAR(100) NULL,
    reference_id NVARCHAR(100) NULL,
    title NVARCHAR(255) NULL,
    body NVARCHAR(MAX) NULL,
    is_read BIT DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    read_at DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION,
    CONSTRAINT fk_notifications_actor FOREIGN KEY (actor_id) REFERENCES dbo.users(id) ON DELETE SET NULL
  );
  CREATE INDEX idx_notifications_user ON dbo.notifications(user_id, is_read, created_at DESC);
END
GO

IF OBJECT_ID('dbo.user_devices','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_devices (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    device_uuid NVARCHAR(200) NULL,
    device_model NVARCHAR(200) NULL,
    os NVARCHAR(100) NULL,
    os_version NVARCHAR(50) NULL,
    app_version NVARCHAR(50) NULL,
    push_token NVARCHAR(500) NULL,
    last_seen DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET(),
    is_active BIT DEFAULT 1,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_devices_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE NO ACTION
  );
  CREATE INDEX idx_user_devices_user ON dbo.user_devices(user_id);
END
GO

IF OBJECT_ID('dbo.audit_logs','U') IS NULL
BEGIN
  CREATE TABLE dbo.audit_logs (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    entity_type NVARCHAR(100),
    entity_id NVARCHAR(100),
    action NVARCHAR(50),
    performed_by UNIQUEIDENTIFIER NULL,
    user_agent NVARCHAR(500) NULL,
    ip_address NVARCHAR(45) NULL,
    payload NVARCHAR(MAX) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL,
    CONSTRAINT fk_audit_performed_by FOREIGN KEY (performed_by) REFERENCES dbo.users(id) ON DELETE SET NULL
  );
  CREATE INDEX idx_audit_created ON dbo.audit_logs(created_at);
END
GO

IF OBJECT_ID('dbo.jobs','U') IS NULL
BEGIN
  CREATE TABLE dbo.jobs (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    topic NVARCHAR(100) NOT NULL,
    payload NVARCHAR(MAX) NOT NULL,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    run_at DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET(),
    locked_until DATETIMEOFFSET NULL,
    locked_by NVARCHAR(100) NULL,
    status NVARCHAR(20) DEFAULT 'pending',
    error_message NVARCHAR(MAX) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    completed_at DATETIMEOFFSET NULL,
    is_deleted BIT NOT NULL DEFAULT 0,
    deleted_at DATETIMEOFFSET NULL
  );
  CREATE INDEX idx_jobs_status_runat ON dbo.jobs(status, run_at);
END
GO


IF OBJECT_ID('dbo.refresh_tokens','U') IS NULL
BEGIN
  CREATE TABLE dbo.refresh_tokens (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    token_hash NVARCHAR(256) NOT NULL,
    device_info NVARCHAR(1000) NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT SYSDATETIMEOFFSET(),
    expires_at DATETIMEOFFSET NOT NULL,
    is_revoked BIT NOT NULL DEFAULT 0,
    CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES dbo.users(id) ON DELETE CASCADE
  );
  CREATE INDEX idx_refresh_tokens_user ON dbo.refresh_tokens(user_id);
  CREATE INDEX idx_refresh_tokens_hash ON dbo.refresh_tokens(token_hash);
END
GO