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
