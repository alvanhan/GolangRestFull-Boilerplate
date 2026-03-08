-- =============================================================================
-- Migration: 001_init
-- Description: Initial schema — extensions, enum types, all tables,
--              indexes, and update_updated_at trigger.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Extensions
-- -----------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- uuid_generate_v4() helper
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- trigram similarity / GIN FTS
CREATE EXTENSION IF NOT EXISTS "btree_gin";    -- allows GIN on scalar columns

-- -----------------------------------------------------------------------------
-- Enum types
-- -----------------------------------------------------------------------------
CREATE TYPE user_role AS ENUM (
    'super_admin',
    'admin',
    'manager',
    'editor',
    'viewer'
);

CREATE TYPE user_status AS ENUM (
    'active',
    'inactive',
    'banned'
);

CREATE TYPE file_status AS ENUM (
    'pending',
    'processing',
    'ready',
    'error',
    'deleted'
);

CREATE TYPE resource_type AS ENUM (
    'file',
    'folder'
);

CREATE TYPE permission_action AS ENUM (
    'read',
    'write',
    'delete',
    'share',
    'download',
    'upload',
    'manage_permissions'
);

-- -----------------------------------------------------------------------------
-- Trigger helper — keeps updated_at in sync automatically
-- -----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER
LANGUAGE plpgsql AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- -----------------------------------------------------------------------------
-- Table: users
-- -----------------------------------------------------------------------------
CREATE TABLE users (
    id                  UUID         NOT NULL DEFAULT gen_random_uuid(),
    email               VARCHAR(255) NOT NULL,
    username            VARCHAR(100) NOT NULL,
    full_name           VARCHAR(255) NOT NULL,
    password_hash       TEXT         NOT NULL,
    role                user_role    NOT NULL DEFAULT 'viewer',
    status              user_status  NOT NULL DEFAULT 'active',
    avatar              TEXT,
    storage_quota       BIGINT       NOT NULL DEFAULT 10737418240, -- 10 GB
    storage_used        BIGINT       NOT NULL DEFAULT 0,
    last_login_at       TIMESTAMPTZ,
    last_login_ip       VARCHAR(45),
    email_verified      BOOLEAN      NOT NULL DEFAULT FALSE,
    two_factor_enabled  BOOLEAN      NOT NULL DEFAULT FALSE,
    metadata            JSONB        NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,

    CONSTRAINT pk_users              PRIMARY KEY (id),
    CONSTRAINT uq_users_email        UNIQUE (email),
    CONSTRAINT uq_users_username     UNIQUE (username),
    CONSTRAINT chk_users_storage_quota CHECK (storage_quota > 0),
    CONSTRAINT chk_users_storage_used  CHECK (storage_used >= 0)
);

CREATE INDEX idx_users_email      ON users (email);
CREATE INDEX idx_users_username   ON users (username);
CREATE INDEX idx_users_role       ON users (role);
CREATE INDEX idx_users_status     ON users (status);
CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: refresh_tokens
-- -----------------------------------------------------------------------------
CREATE TABLE refresh_tokens (
    id          UUID         NOT NULL DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    token_hash  TEXT         NOT NULL,
    user_agent  TEXT,
    ip_address  VARCHAR(45),
    expires_at  TIMESTAMPTZ  NOT NULL,
    revoked     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_refresh_tokens       PRIMARY KEY (id),
    CONSTRAINT uq_refresh_tokens_hash  UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
CREATE INDEX idx_refresh_tokens_revoked    ON refresh_tokens (revoked) WHERE revoked = FALSE;

-- -----------------------------------------------------------------------------
-- Table: folders
-- -----------------------------------------------------------------------------
CREATE TABLE folders (
    id           UUID         NOT NULL DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    owner_id     UUID         NOT NULL,
    parent_id    UUID,
    path         TEXT         NOT NULL,  -- materialised path e.g. "/rootId/childId"
    is_root      BOOLEAN      NOT NULL DEFAULT FALSE,
    is_shared    BOOLEAN      NOT NULL DEFAULT FALSE,
    color        VARCHAR(20),
    icon         VARCHAR(50),
    size         BIGINT       NOT NULL DEFAULT 0,
    file_count   BIGINT       NOT NULL DEFAULT 0,
    folder_count BIGINT       NOT NULL DEFAULT 0,
    metadata     JSONB        NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

    CONSTRAINT pk_folders              PRIMARY KEY (id),
    CONSTRAINT fk_folders_owner        FOREIGN KEY (owner_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_folders_parent       FOREIGN KEY (parent_id)
        REFERENCES folders (id) ON DELETE CASCADE,
    CONSTRAINT chk_folders_size         CHECK (size >= 0),
    CONSTRAINT chk_folders_file_count   CHECK (file_count >= 0),
    CONSTRAINT chk_folders_folder_count CHECK (folder_count >= 0)
);

CREATE INDEX idx_folders_owner_id   ON folders (owner_id);
CREATE INDEX idx_folders_parent_id  ON folders (parent_id);
CREATE INDEX idx_folders_path       ON folders (path text_pattern_ops);
CREATE INDEX idx_folders_deleted_at ON folders (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_folders_is_root    ON folders (is_root)    WHERE is_root = TRUE;
CREATE INDEX idx_folders_name_trgm  ON folders USING GIN (name gin_trgm_ops);

CREATE TRIGGER trg_folders_updated_at
    BEFORE UPDATE ON folders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: files
-- -----------------------------------------------------------------------------
CREATE TABLE files (
    id               UUID         NOT NULL DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    original_name    VARCHAR(255) NOT NULL,
    extension        VARCHAR(50)  NOT NULL,
    mime_type        VARCHAR(255) NOT NULL,
    size             BIGINT       NOT NULL,
    checksum         VARCHAR(64)  NOT NULL,  -- SHA-256 hex digest (64 chars)
    storage_key      TEXT         NOT NULL,  -- MinIO object key
    storage_bucket   VARCHAR(255) NOT NULL,
    folder_id        UUID,                   -- NULL means owner root
    owner_id         UUID         NOT NULL,
    version          INT          NOT NULL DEFAULT 1,
    status           file_status  NOT NULL DEFAULT 'pending',
    is_encrypted     BOOLEAN      NOT NULL DEFAULT FALSE,
    is_public        BOOLEAN      NOT NULL DEFAULT FALSE,
    download_count   BIGINT       NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    expires_at       TIMESTAMPTZ,
    tags             TEXT[]       NOT NULL DEFAULT '{}',
    metadata         JSONB        NOT NULL DEFAULT '{}',
    thumbnail_key    TEXT,
    description      TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,

    CONSTRAINT pk_files             PRIMARY KEY (id),
    CONSTRAINT uq_files_storage_key UNIQUE (storage_key),
    CONSTRAINT fk_files_owner       FOREIGN KEY (owner_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_files_folder      FOREIGN KEY (folder_id)
        REFERENCES folders (id) ON DELETE SET NULL,
    CONSTRAINT chk_files_size           CHECK (size > 0),
    CONSTRAINT chk_files_version        CHECK (version >= 1),
    CONSTRAINT chk_files_download_count CHECK (download_count >= 0)
);

CREATE INDEX idx_files_owner_id    ON files (owner_id);
CREATE INDEX idx_files_folder_id   ON files (folder_id);
CREATE INDEX idx_files_status      ON files (status);
CREATE INDEX idx_files_mime_type   ON files (mime_type);
CREATE INDEX idx_files_deleted_at  ON files (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_files_is_public   ON files (is_public)  WHERE is_public = TRUE;
CREATE INDEX idx_files_expires_at  ON files (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_created_at  ON files (created_at DESC);

-- GIN index for full-text search on name and description
CREATE INDEX idx_files_name_fts ON files USING GIN (
    to_tsvector('english', name || ' ' || COALESCE(description, ''))
);

-- GIN index for tag array containment queries (e.g. tags @> ARRAY['invoice'])
CREATE INDEX idx_files_tags_gin ON files USING GIN (tags);

-- Trigram index for ILIKE / similarity searches on file name
CREATE INDEX idx_files_name_trgm ON files USING GIN (name gin_trgm_ops);

CREATE TRIGGER trg_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: file_versions
-- -----------------------------------------------------------------------------
CREATE TABLE file_versions (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    file_id       UUID        NOT NULL,
    version       INT         NOT NULL,
    storage_key   TEXT        NOT NULL,
    size          BIGINT      NOT NULL,
    checksum      VARCHAR(64) NOT NULL,
    changed_by_id UUID        NOT NULL,
    change_note   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_file_versions                 PRIMARY KEY (id),
    CONSTRAINT uq_file_versions_file_version    UNIQUE (file_id, version),
    CONSTRAINT fk_file_versions_file            FOREIGN KEY (file_id)
        REFERENCES files (id) ON DELETE CASCADE,
    CONSTRAINT fk_file_versions_changed_by      FOREIGN KEY (changed_by_id)
        REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_file_versions_size    CHECK (size > 0),
    CONSTRAINT chk_file_versions_version CHECK (version >= 1)
);

CREATE INDEX idx_file_versions_file_id    ON file_versions (file_id);
CREATE INDEX idx_file_versions_created_at ON file_versions (created_at DESC);

-- -----------------------------------------------------------------------------
-- Table: file_chunks  (tracks progress of chunked / multipart uploads)
-- -----------------------------------------------------------------------------
CREATE TABLE file_chunks (
    id           UUID         NOT NULL DEFAULT gen_random_uuid(),
    upload_id    VARCHAR(255) NOT NULL,   -- client-generated session ID
    file_key     TEXT         NOT NULL,   -- intended final storage key
    chunk_index  INT          NOT NULL,   -- 0-based chunk index
    chunk_size   BIGINT       NOT NULL,
    total_chunks INT          NOT NULL,
    storage_key  TEXT         NOT NULL,   -- temporary MinIO object key
    checksum     VARCHAR(64)  NOT NULL,
    uploaded_by  UUID         NOT NULL,
    expires_at   TIMESTAMPTZ  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_file_chunks                  PRIMARY KEY (id),
    CONSTRAINT uq_file_chunks_upload_chunk     UNIQUE (upload_id, chunk_index),
    CONSTRAINT fk_file_chunks_user             FOREIGN KEY (uploaded_by)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT chk_file_chunks_chunk_index    CHECK (chunk_index >= 0),
    CONSTRAINT chk_file_chunks_total_chunks   CHECK (total_chunks > 0),
    CONSTRAINT chk_file_chunks_chunk_size     CHECK (chunk_size > 0)
);

-- Composite index matches the most common query pattern: lookup by upload session
CREATE INDEX idx_file_chunks_upload_id_chunk_index ON file_chunks (upload_id, chunk_index);
CREATE INDEX idx_file_chunks_uploaded_by           ON file_chunks (uploaded_by);
CREATE INDEX idx_file_chunks_expires_at            ON file_chunks (expires_at);

-- -----------------------------------------------------------------------------
-- Table: permissions
-- -----------------------------------------------------------------------------
CREATE TABLE permissions (
    id            UUID              NOT NULL DEFAULT gen_random_uuid(),
    resource_id   UUID              NOT NULL,
    resource_type resource_type     NOT NULL,
    user_id       UUID              NOT NULL,
    action        permission_action NOT NULL,
    granted_by_id UUID              NOT NULL,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_permissions                       PRIMARY KEY (id),
    CONSTRAINT uq_permissions_resource_user_action  UNIQUE (resource_id, resource_type, user_id, action),
    CONSTRAINT fk_permissions_user                  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_permissions_granted_by            FOREIGN KEY (granted_by_id)
        REFERENCES users (id) ON DELETE RESTRICT
);

-- Covers the most common access-check query
CREATE INDEX idx_permissions_resource         ON permissions (resource_id, resource_type);
CREATE INDEX idx_permissions_user_id          ON permissions (user_id);
CREATE INDEX idx_permissions_resource_user    ON permissions (resource_id, resource_type, user_id, action);
CREATE INDEX idx_permissions_expires_at       ON permissions (expires_at) WHERE expires_at IS NOT NULL;

CREATE TRIGGER trg_permissions_updated_at
    BEFORE UPDATE ON permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: share_links
-- -----------------------------------------------------------------------------
CREATE TABLE share_links (
    id            UUID              NOT NULL DEFAULT gen_random_uuid(),
    token         VARCHAR(255)      NOT NULL,
    resource_id   UUID              NOT NULL,
    resource_type resource_type     NOT NULL,
    created_by_id UUID              NOT NULL,
    action        permission_action NOT NULL DEFAULT 'read',
    password      TEXT,                         -- bcrypt hash of optional password
    expires_at    TIMESTAMPTZ,
    max_uses      INT,
    use_count     INT               NOT NULL DEFAULT 0,
    is_active     BOOLEAN           NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_share_links            PRIMARY KEY (id),
    CONSTRAINT uq_share_links_token      UNIQUE (token),
    CONSTRAINT fk_share_links_created_by FOREIGN KEY (created_by_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT chk_share_links_use_count CHECK (use_count >= 0),
    CONSTRAINT chk_share_links_max_uses  CHECK (max_uses IS NULL OR max_uses > 0)
);

CREATE INDEX idx_share_links_resource_id   ON share_links (resource_id, resource_type);
CREATE INDEX idx_share_links_created_by_id ON share_links (created_by_id);
CREATE INDEX idx_share_links_is_active     ON share_links (is_active) WHERE is_active = TRUE;
CREATE INDEX idx_share_links_expires_at    ON share_links (expires_at) WHERE expires_at IS NOT NULL;

CREATE TRIGGER trg_share_links_updated_at
    BEFORE UPDATE ON share_links
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: audit_logs
-- Append-only — no updated_at, no soft-delete.  For very high-volume
-- environments consider partitioning by month:
--   PARTITION BY RANGE (created_at)
-- and creating monthly child tables via a scheduled task.
-- -----------------------------------------------------------------------------
CREATE TABLE audit_logs (
    id            UUID         NOT NULL DEFAULT gen_random_uuid(),
    user_id       UUID,                   -- NULL for system-initiated actions
    action        VARCHAR(50)  NOT NULL,
    resource_id   UUID,
    resource_type VARCHAR(20),
    resource_name VARCHAR(255),
    ip_address    VARCHAR(45)  NOT NULL DEFAULT '',
    user_agent    TEXT         NOT NULL DEFAULT '',
    details       JSONB        NOT NULL DEFAULT '{}',
    old_values    JSONB        NOT NULL DEFAULT '{}',
    new_values    JSONB        NOT NULL DEFAULT '{}',
    status        VARCHAR(10)  NOT NULL DEFAULT 'success',
    error_message TEXT,
    duration      BIGINT       NOT NULL DEFAULT 0,  -- request duration in ms
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_audit_logs       PRIMARY KEY (id),
    CONSTRAINT fk_audit_logs_user  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT chk_audit_logs_status    CHECK (status IN ('success', 'failed')),
    CONSTRAINT chk_audit_logs_duration  CHECK (duration >= 0)
);

-- Time-based queries (dashboards, retention cleanup)
CREATE INDEX idx_audit_logs_created_at    ON audit_logs (created_at DESC);
-- Filtering by actor
CREATE INDEX idx_audit_logs_user_id       ON audit_logs (user_id) WHERE user_id IS NOT NULL;
-- Filtering by affected resource
CREATE INDEX idx_audit_logs_resource      ON audit_logs (resource_id, resource_type)
    WHERE resource_id IS NOT NULL;
-- Filtering by action type
CREATE INDEX idx_audit_logs_action        ON audit_logs (action);
-- JSONB containment queries on details (e.g. details @> '{"tag": "critical"}')
CREATE INDEX idx_audit_logs_details_gin   ON audit_logs USING GIN (details jsonb_path_ops);

-- -----------------------------------------------------------------------------
-- Table: notifications
-- -----------------------------------------------------------------------------
CREATE TABLE notifications (
    id            UUID         NOT NULL DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL,
    type          VARCHAR(50)  NOT NULL,
    title         VARCHAR(255) NOT NULL,
    message       TEXT         NOT NULL,
    resource_id   UUID,
    resource_type VARCHAR(20),
    is_read       BOOLEAN      NOT NULL DEFAULT FALSE,
    read_at       TIMESTAMPTZ,
    metadata      JSONB        NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_notifications       PRIMARY KEY (id),
    CONSTRAINT fk_notifications_user  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT chk_notifications_read_at CHECK (
        (is_read = FALSE AND read_at IS NULL) OR
        (is_read = TRUE  AND read_at IS NOT NULL)
    )
);

-- Inbox query: fetch unread notifications for a user, newest first
CREATE INDEX idx_notifications_user_unread   ON notifications (user_id, created_at DESC)
    WHERE is_read = FALSE;
-- Full inbox (read + unread)
CREATE INDEX idx_notifications_user_id       ON notifications (user_id, created_at DESC);
-- Bulk operations on resource-linked notifications
CREATE INDEX idx_notifications_resource      ON notifications (resource_id, resource_type)
    WHERE resource_id IS NOT NULL;
