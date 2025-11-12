USE GatherUpDB;
GO

-- chats
IF OBJECT_ID('dbo.chats','U') IS NULL
BEGIN
 CREATE TABLE dbo.chats (
   id BIGINT IDENTITY(1,1) PRIMARY KEY,
   uuid CHAR(36) NOT NULL UNIQUE,
   title NVARCHAR(255) NULL, -- optional for group
   is_group BIT DEFAULT 0,
   created_by CHAR(36) NULL,
   created_at DATETIME2 DEFAULT SYSUTCDATETIME()
 );
 CREATE INDEX idx_chats_created_at ON dbo.chats(created_at);
END
GO

-- participants
IF OBJECT_ID('dbo.chat_participants','U') IS NULL
BEGIN
 CREATE TABLE dbo.chat_participants (
   id BIGINT IDENTITY(1,1) PRIMARY KEY,
   chat_uuid CHAR(36) NOT NULL,
   user_uid CHAR(36) NOT NULL,
   joined_at DATETIME2 DEFAULT SYSUTCDATETIME(),
   is_admin BIT DEFAULT 0
 );
 CREATE INDEX idx_part_chat_user ON dbo.chat_participants(chat_uuid, user_uid);
END
GO

-- messages
IF OBJECT_ID('dbo.messages','U') IS NULL
BEGIN
 CREATE TABLE dbo.messages (
   id BIGINT IDENTITY(1,1) PRIMARY KEY,
   uuid CHAR(36) NOT NULL UNIQUE,
   chat_uuid CHAR(36) NOT NULL,
   sender_uid CHAR(36) NOT NULL,
   body NVARCHAR(MAX) NULL,
   media_url NVARCHAR(1024) NULL,
   kind NVARCHAR(50) DEFAULT 'text', -- text/image/system/notice
   status NVARCHAR(50) DEFAULT 'sent', -- sent/delivered/read/failed
   created_at DATETIME2 DEFAULT SYSUTCDATETIME()
 );
 CREATE INDEX idx_messages_chat_created ON dbo.messages(chat_uuid, created_at DESC);
END
GO





------- New Table Structure

-- Use your DB
USE GatherUpDB;
GO

----------------------------------------------------------------------
-- 1) Utility / lookup tables
-- These are small lookups for statuses, kinds, visibility, reaction types.
----------------------------------------------------------------------
IF OBJECT_ID('dbo.visibility_types','U') IS NULL
BEGIN
  CREATE TABLE dbo.visibility_types (
    id TINYINT PRIMARY KEY, -- small table, fixed ids (0=private,1=friends,2=public,3=group-members)
    code NVARCHAR(50) NOT NULL UNIQUE,
    description NVARCHAR(255) NULL
  );
  INSERT INTO dbo.visibility_types(id, code, description) VALUES
    (0,'private','Visible to selected users only'),
    (1,'contacts','Visible to contacts/friends'),
    (2,'public','Visible to everyone'),
    (3,'group','Visible to group members');
END
GO

IF OBJECT_ID('dbo.reaction_types','U') IS NULL
BEGIN
  CREATE TABLE dbo.reaction_types (
    id TINYINT PRIMARY KEY, -- e.g., 1=like,2=love,3=haha...
    code NVARCHAR(50) NOT NULL UNIQUE,
    icon NVARCHAR(50) NULL
  );
  INSERT INTO dbo.reaction_types(id, code, icon) VALUES
    (1,'like','thumbs-up'),
    (2,'love','heart'),
    (3,'laugh','laugh');
END
GO

----------------------------------------------------------------------
-- 2) Users & Auth
-- Users core, profile, auth methods, sessions.
----------------------------------------------------------------------
IF OBJECT_ID('dbo.users','U') IS NULL
BEGIN
  CREATE TABLE dbo.users (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(), -- use GUIDs for public references
    username NVARCHAR(100) NOT NULL UNIQUE, -- login handle
    email NVARCHAR(320) NOT NULL UNIQUE,
    is_email_verified BIT DEFAULT 0,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at DATETIME2 NULL,
    is_active BIT DEFAULT 1,
    -- soft-delete
    deleted_at DATETIME2 NULL,
    -- concurrency token
    rv ROWVERSION NOT NULL
  );
  CREATE INDEX idx_users_created_at ON dbo.users(created_at);
END
GO

-- Authentication methods (passwords / oauth / device)
IF OBJECT_ID('dbo.user_credentials','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_credentials (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    credential_type NVARCHAR(50) NOT NULL, -- e.g., 'password','google','facebook','device'
    credential_identifier NVARCHAR(512) NULL, -- e.g., oauth subject
    password_hash NVARCHAR(1024) NULL, -- only used for credential_type='password'
    salt VARBINARY(64) NULL,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_usercred_user ON dbo.user_credentials(user_id);
END
GO

-- Sessions / tokens for login sessions
IF OBJECT_ID('dbo.user_sessions','U') IS NULL
BEGIN
  CREATE TABLE dbo.user_sessions (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    device_info NVARCHAR(512) NULL,
    ip_address NVARCHAR(45) NULL,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    expires_at DATETIME2 NULL,
    is_revoked BIT DEFAULT 0
  );
  CREATE INDEX idx_sessions_user ON dbo.user_sessions(user_id);
END
GO

----------------------------------------------------------------------
-- 3) Contacts / friends
-- Contact list / follow / block — flexible to support 1-way follow or mutual friend.
----------------------------------------------------------------------
IF OBJECT_ID('dbo.contacts','U') IS NULL
BEGIN
  CREATE TABLE dbo.contacts (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE, -- owner of the contact list
    contact_user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    relation_type NVARCHAR(50) NOT NULL DEFAULT 'contact', -- 'contact','blocked','following','friend_request'
    status NVARCHAR(50) NOT NULL DEFAULT 'active', -- 'active','pending','rejected'
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_contacts_user_contact UNIQUE (user_id, contact_user_id)
  );
  CREATE INDEX idx_contacts_user ON dbo.contacts(user_id);
END
GO

----------------------------------------------------------------------
-- 4) Posts / Media / Post visibility / Post recipients
-- Posts can be public or private. For private, we store explicit recipients.
----------------------------------------------------------------------
IF OBJECT_ID('dbo.posts','U') IS NULL
BEGIN
  CREATE TABLE dbo.posts (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    author_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    title NVARCHAR(255) NULL, -- optional for long posts
    body NVARCHAR(MAX) NULL,
    kind NVARCHAR(50) NOT NULL DEFAULT 'text', -- 'text','image','link','event' etc
    visibility_id TINYINT NOT NULL DEFAULT 2 REFERENCES dbo.visibility_types(id), -- refer to lookup
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at DATETIME2 NULL,
    deleted_at DATETIME2 NULL, -- soft delete
    rv ROWVERSION NOT NULL
  );
  CREATE INDEX idx_posts_author_created ON dbo.posts(author_id, created_at DESC);
  -- Full-text index recommended on body for search (configure separately)
END
GO

-- Attach media items (images, videos) to posts (multiple media per post)
IF OBJECT_ID('dbo.post_media','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_media (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    media_url NVARCHAR(2048) NOT NULL, -- CDN URL or blob path
    media_type NVARCHAR(50) NULL, -- 'image','video','thumbnail'
    sort_order INT DEFAULT 0,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_post_media_post ON dbo.post_media(post_id);
END
GO

-- For private posts / custom recipients list
IF OBJECT_ID('dbo.post_recipients','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_recipients (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    recipient_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_post_recipient UNIQUE(post_id, recipient_id)
  );
  CREATE INDEX idx_post_recip_on_recipient ON dbo.post_recipients(recipient_id);
END
GO

----------------------------------------------------------------------
-- 5) Post interactions: views, reactions, shares, comments
-- Keep individual interaction records for uniqueness & audit, and aggregate counters separately.
----------------------------------------------------------------------
-- Track individual views (unique views per user or anonymous token)
IF OBJECT_ID('dbo.post_views','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_views (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    viewer_id UNIQUEIDENTIFIER NULL, -- null if anonymous
    viewer_ip NVARCHAR(45) NULL,
    viewed_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    -- unique constraint to prevent double counting (if you want unique per user)
    CONSTRAINT ux_post_views_unique_user UNIQUE (post_id, viewer_id) WHERE viewer_id IS NOT NULL
  );
  CREATE INDEX idx_post_views_post ON dbo.post_views(post_id);
END
GO

-- Reactions (likes) generalized
IF OBJECT_ID('dbo.post_reactions','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_reactions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    reaction_type_id TINYINT NOT NULL REFERENCES dbo.reaction_types(id),
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_post_reaction_unique UNIQUE (post_id, user_id, reaction_type_id)
  );
  CREATE INDEX idx_post_reactions_post ON dbo.post_reactions(post_id);
END
GO

-- Shares: who shared and optional comment
IF OBJECT_ID('dbo.post_shares','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_shares (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    sharer_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    share_comment NVARCHAR(1000) NULL,
    shared_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_post_shares_post ON dbo.post_shares(post_id);
END
GO

-- Comments on posts (nested optional via parent_comment_id)
IF OBJECT_ID('dbo.comments','U') IS NULL
BEGIN
  CREATE TABLE dbo.comments (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    post_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.posts(id) ON DELETE CASCADE,
    author_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    parent_comment_id BIGINT NULL REFERENCES dbo.comments(id) ON DELETE CASCADE,
    body NVARCHAR(MAX) NOT NULL,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    updated_at DATETIME2 NULL,
    deleted_at DATETIME2 NULL
  );
  CREATE INDEX idx_comments_post ON dbo.comments(post_id, created_at DESC);
END
GO

-- Comment reactions (likes on comment)
IF OBJECT_ID('dbo.comment_reactions','U') IS NULL
BEGIN
  CREATE TABLE dbo.comment_reactions (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    comment_id BIGINT NOT NULL REFERENCES dbo.comments(id) ON DELETE CASCADE,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    reaction_type_id TINYINT NOT NULL REFERENCES dbo.reaction_types(id),
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_comment_reaction UNIQUE (comment_id, user_id, reaction_type_id)
  );
  CREATE INDEX idx_comment_react_comment ON dbo.comment_reactions(comment_id);
END
GO

----------------------------------------------------------------------
-- 6) Chats and messages (updated, robust)
-- Chat = 1:1 or group, participants, per-recipient message status
----------------------------------------------------------------------
IF OBJECT_ID('dbo.chats','U') IS NULL
BEGIN
  CREATE TABLE dbo.chats (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    uuid AS CONVERT(NVARCHAR(36), id) PERSISTED, -- string form if needed
    title NVARCHAR(255) NULL, -- group title
    is_group BIT DEFAULT 0,
    created_by UNIQUEIDENTIFIER NULL REFERENCES dbo.users(id),
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    deleted_at DATETIME2 NULL
  );
  CREATE INDEX idx_chats_created_at ON dbo.chats(created_at);
END
GO

-- Participants in a chat
IF OBJECT_ID('dbo.chat_participants','U') IS NULL
BEGIN
  CREATE TABLE dbo.chat_participants (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    chat_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.chats(id) ON DELETE CASCADE,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    joined_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    left_at DATETIME2 NULL,
    is_admin BIT DEFAULT 0,
    CONSTRAINT ux_chat_participant UNIQUE (chat_id, user_id)
  );
  CREATE INDEX idx_chat_part_on_user ON dbo.chat_participants(chat_id, user_id);
END
GO

-- Messages table
IF OBJECT_ID('dbo.messages','U') IS NULL
BEGIN
  CREATE TABLE dbo.messages (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    chat_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.chats(id) ON DELETE CASCADE,
    sender_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    body NVARCHAR(MAX) NULL,
    kind NVARCHAR(50) DEFAULT 'text', -- 'text','image','system','notice'
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    edited_at DATETIME2 NULL,
    deleted_at DATETIME2 NULL,
    rv ROWVERSION NOT NULL
  );
  CREATE INDEX idx_messages_chat_created ON dbo.messages(chat_id, created_at DESC);
END
GO

-- Message media (attachments)
IF OBJECT_ID('dbo.message_media','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_media (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.messages(id) ON DELETE CASCADE,
    media_url NVARCHAR(2048) NOT NULL,
    media_type NVARCHAR(50) NULL,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_message_media_msg ON dbo.message_media(message_id);
END
GO

-- Per-recipient message status (delivered/read/failed)
IF OBJECT_ID('dbo.message_status','U') IS NULL
BEGIN
  CREATE TABLE dbo.message_status (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    message_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.messages(id) ON DELETE CASCADE,
    recipient_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    status NVARCHAR(50) NOT NULL DEFAULT 'sent', -- 'sent','delivered','read','failed'
    status_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_msg_status_unique UNIQUE (message_id, recipient_id)
  );
  CREATE INDEX idx_msg_status_message ON dbo.message_status(message_id);
END
GO

----------------------------------------------------------------------
-- 7) Games / Tournaments / Matches
-- Catalog of games (cricket, chess), tournaments, signups, matches, results.
----------------------------------------------------------------------
IF OBJECT_ID('dbo.games_catalog','U') IS NULL
BEGIN
  CREATE TABLE dbo.games_catalog (
    id INT IDENTITY(1,1) PRIMARY KEY,
    code NVARCHAR(50) NOT NULL UNIQUE, -- 'cricket','chess'
    display_name NVARCHAR(100) NOT NULL,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

-- Tournament or Game event created by a user
IF OBJECT_ID('dbo.tournaments','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournaments (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    creator_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE SET NULL,
    game_id INT NOT NULL REFERENCES dbo.games_catalog(id),
    title NVARCHAR(255) NOT NULL,
    description NVARCHAR(MAX) NULL,
    place NVARCHAR(255) NULL,
    start_at DATETIME2 NULL,
    end_at DATETIME2 NULL,
    is_public BIT DEFAULT 1, -- public or private event
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    status NVARCHAR(50) DEFAULT 'scheduled', -- scheduled,ongoing,completed,cancelled
    rv ROWVERSION NOT NULL
  );
  CREATE INDEX idx_tournament_game ON dbo.tournaments(game_id);
  CREATE INDEX idx_tournament_start ON dbo.tournaments(start_at);
END
GO

-- Tournament participants (teams or users)
IF OBJECT_ID('dbo.tournament_participants','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_participants (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tournament_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    user_id UNIQUEIDENTIFIER NULL REFERENCES dbo.users(id), -- optional if participant is a team entity
    team_name NVARCHAR(255) NULL,
    joined_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    accepted BIT DEFAULT 0, -- creator needs to accept join requests
    CONSTRAINT ux_tourn_part_unique UNIQUE (tournament_id, user_id, team_name)
  );
  CREATE INDEX idx_tourn_part_tourn ON dbo.tournament_participants(tournament_id);
END
GO

-- Matches within tournament (rounds)
IF OBJECT_ID('dbo.matches','U') IS NULL
BEGIN
  CREATE TABLE dbo.matches (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWSEQUENTIALID(),
    tournament_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    round INT NULL,
    participant_a_id BIGINT NULL REFERENCES dbo.tournament_participants(id),
    participant_b_id BIGINT NULL REFERENCES dbo.tournament_participants(id),
    scheduled_at DATETIME2 NULL,
    place NVARCHAR(255) NULL,
    status NVARCHAR(50) DEFAULT 'scheduled', -- scheduled,ongoing,finished
    result JSON NULL, -- small JSON summary (scores etc). SQL Server 2016+ supports JSON functions.
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_matches_tourn ON dbo.matches(tournament_id);
END
GO

-- Scoreboard / aggregated results (can be generated by ETL/SP)
IF OBJECT_ID('dbo.tournament_scores','U') IS NULL
BEGIN
  CREATE TABLE dbo.tournament_scores (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    tournament_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.tournaments(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL REFERENCES dbo.tournament_participants(id) ON DELETE CASCADE,
    played INT DEFAULT 0,
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    draws INT DEFAULT 0,
    points INT DEFAULT 0,
    last_updated DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT ux_tourn_score_unique UNIQUE (tournament_id, participant_id)
  );
  CREATE INDEX idx_tourn_scores ON dbo.tournament_scores(tournament_id, points DESC);
END
GO

----------------------------------------------------------------------
-- 8) Notifications & Invitations (for invites to games, chats)
----------------------------------------------------------------------
IF OBJECT_ID('dbo.notifications','U') IS NULL
BEGIN
  CREATE TABLE dbo.notifications (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL REFERENCES dbo.users(id) ON DELETE CASCADE,
    actor_id UNIQUEIDENTIFIER NULL REFERENCES dbo.users(id),
    kind NVARCHAR(100) NOT NULL, -- 'invite','comment','like','match_update'
    reference_type NVARCHAR(100) NULL, -- e.g., 'post','tournament','match','chat'
    reference_id NVARCHAR(100) NULL, -- id in text (can be guid/int)
    body NVARCHAR(MAX) NULL,
    is_read BIT DEFAULT 0,
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_notifications_user ON dbo.notifications(user_id, is_read);
END
GO

----------------------------------------------------------------------
-- 9) Audit / admin logs (optional)
----------------------------------------------------------------------
IF OBJECT_ID('dbo.audit_logs','U') IS NULL
BEGIN
  CREATE TABLE dbo.audit_logs (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    entity_type NVARCHAR(100),
    entity_id NVARCHAR(100),
    action NVARCHAR(50), -- 'create','update','delete'
    performed_by UNIQUEIDENTIFIER NULL REFERENCES dbo.users(id),
    payload NVARCHAR(MAX) NULL, -- JSON snapshot or diff
    created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
  CREATE INDEX idx_audit_created ON dbo.audit_logs(created_at);
END
GO

----------------------------------------------------------------------
-- 10) Helpful sample stored procedure / atomic counters
-- Example: increment a post view counter safely by inserting into post_views then updating aggregator.
----------------------------------------------------------------------
-- Aggregated counters table (performance: cheaper reads)
IF OBJECT_ID('dbo.post_counters','U') IS NULL
BEGIN
  CREATE TABLE dbo.post_counters (
    post_id UNIQUEIDENTIFIER PRIMARY KEY REFERENCES dbo.posts(id),
    view_count BIGINT DEFAULT 0,
    reaction_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    share_count BIGINT DEFAULT 0,
    last_updated DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
  );
END
GO

-- Example stored proc to record unique view and bump counter
IF OBJECT_ID('dbo.sp_record_post_view','P') IS NULL
BEGIN
EXEC('
CREATE PROCEDURE dbo.sp_record_post_view
  @post_id UNIQUEIDENTIFIER,
  @viewer_id UNIQUEIDENTIFIER = NULL,  -- optional
  @viewer_ip NVARCHAR(45) = NULL
AS
BEGIN
  SET NOCOUNT ON;
  BEGIN TRY
    BEGIN TRAN;

    -- 1) insert view if not exists for this user (unique)
    IF @viewer_id IS NOT NULL
    BEGIN
      IF NOT EXISTS (SELECT 1 FROM dbo.post_views pv WHERE pv.post_id = @post_id AND pv.viewer_id = @viewer_id)
      BEGIN
        INSERT INTO dbo.post_views(post_id, viewer_id, viewer_ip) VALUES(@post_id, @viewer_id, @viewer_ip);
        -- upsert into counters
        MERGE dbo.post_counters AS target
        USING (SELECT @post_id AS post_id) AS src
        ON (target.post_id = src.post_id)
        WHEN MATCHED THEN
          UPDATE SET view_count = target.view_count + 1, last_updated = SYSUTCDATETIME()
        WHEN NOT MATCHED THEN
          INSERT (post_id, view_count, last_updated) VALUES (src.post_id, 1, SYSUTCDATETIME());
      END
    END
    ELSE
    BEGIN
      -- For anonymous viewers, always record and bump (or use IP+day dedupe if you want)
      INSERT INTO dbo.post_views(post_id, viewer_id, viewer_ip) VALUES(@post_id, NULL, @viewer_ip);
      MERGE dbo.post_counters AS target
      USING (SELECT @post_id AS post_id) AS src
      ON (target.post_id = src.post_id)
      WHEN MATCHED THEN
        UPDATE SET view_count = target.view_count + 1, last_updated = SYSUTCDATETIME()
      WHEN NOT MATCHED THEN
        INSERT (post_id, view_count, last_updated) VALUES (src.post_id, 1, SYSUTCDATETIME());
    END

    COMMIT;
  END TRY
  BEGIN CATCH
    IF @@TRANCOUNT > 0 ROLLBACK;
    THROW;
  END CATCH
END
');
END
GO

----------------------------------------------------------------------
-- 11) Useful index suggestions (already included inline but listing)
-- - Index posts by (created_at, author) for feeds.
-- - Index messages by (chat_id, created_at desc) for chat history.
-- - Maintain counters table for fast read of like/view counts.
----------------------------------------------------------------------

