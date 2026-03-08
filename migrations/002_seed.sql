-- =============================================================================
-- Migration: 002_seed
-- Description: Bootstrap seed data — system users, root folders, and reference
--              records required for the application to function out of the box.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Helper: wrap everything in a transaction so a partial failure rolls back
-- -----------------------------------------------------------------------------
BEGIN;

-- -----------------------------------------------------------------------------
-- Super admin user
-- Credentials: superadmin@system.local / Admin@123456
-- Password hash: bcrypt cost=12 of "Admin@123456"
-- Generate a fresh hash for production with:
--   go run -modfile=go.mod ./cmd/tools/genhash Admin@123456
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    username,
    full_name,
    password_hash,
    role,
    status,
    email_verified,
    two_factor_enabled,
    storage_quota,
    storage_used,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'superadmin@system.local',
    'superadmin',
    'System Administrator',
    '$2a$12$fijeEmcL62ExF/WaLXd5ee5.CYt.jOC2Ptr7bMNIcVabN3A4FxyTe',
    'super_admin',
    'active',
    TRUE,
    FALSE,
    107374182400,   -- 100 GB
    0,
    '{"onboarded": true, "system_account": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Admin user
-- Credentials: admin@system.local / Admin@123456
-- Same hash used for demo purposes — change immediately in production.
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    username,
    full_name,
    password_hash,
    role,
    status,
    email_verified,
    two_factor_enabled,
    storage_quota,
    storage_used,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000002',
    'admin@system.local',
    'admin',
    'Platform Administrator',
    '$2a$12$fijeEmcL62ExF/WaLXd5ee5.CYt.jOC2Ptr7bMNIcVabN3A4FxyTe',
    'admin',
    'active',
    TRUE,
    FALSE,
    53687091200,    -- 50 GB
    0,
    '{"onboarded": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Demo manager — useful for integration / end-to-end testing
-- Credentials: manager@demo.local / Admin@123456
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    username,
    full_name,
    password_hash,
    role,
    status,
    email_verified,
    storage_quota,
    storage_used,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000003',
    'manager@demo.local',
    'demo_manager',
    'Demo Manager',
    '$2a$12$fijeEmcL62ExF/WaLXd5ee5.CYt.jOC2Ptr7bMNIcVabN3A4FxyTe',
    'manager',
    'active',
    TRUE,
    10737418240,    -- 10 GB (default)
    0,
    '{"demo_account": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Demo editor
-- Credentials: editor@demo.local / Admin@123456
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    username,
    full_name,
    password_hash,
    role,
    status,
    email_verified,
    storage_quota,
    storage_used,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000004',
    'editor@demo.local',
    'demo_editor',
    'Demo Editor',
    '$2a$12$fijeEmcL62ExF/WaLXd5ee5.CYt.jOC2Ptr7bMNIcVabN3A4FxyTe',
    'editor',
    'active',
    TRUE,
    10737418240,
    0,
    '{"demo_account": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Demo viewer (read-only)
-- Credentials: viewer@demo.local / Admin@123456
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id,
    email,
    username,
    full_name,
    password_hash,
    role,
    status,
    email_verified,
    storage_quota,
    storage_used,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000005',
    'viewer@demo.local',
    'demo_viewer',
    'Demo Viewer',
    '$2a$12$fijeEmcL62ExF/WaLXd5ee5.CYt.jOC2Ptr7bMNIcVabN3A4FxyTe',
    'viewer',
    'active',
    TRUE,
    5368709120,     -- 5 GB
    0,
    '{"demo_account": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Root folders — one per seed user.
-- Each user gets a personal root folder (parent_id = NULL, is_root = TRUE).
-- The path convention is "/<folder_uuid>" for root-level folders.
-- -----------------------------------------------------------------------------

-- Super admin root folder
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    metadata
) VALUES (
    '10000000-0000-0000-0000-000000000001',
    'My Files',
    'Personal root folder for System Administrator',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    '/10000000-0000-0000-0000-000000000001',
    TRUE,
    FALSE,
    '{}'::jsonb
);

-- Admin root folder
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    metadata
) VALUES (
    '10000000-0000-0000-0000-000000000002',
    'My Files',
    'Personal root folder for Platform Administrator',
    '00000000-0000-0000-0000-000000000002',
    NULL,
    '/10000000-0000-0000-0000-000000000002',
    TRUE,
    FALSE,
    '{}'::jsonb
);

-- Manager root folder
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    metadata
) VALUES (
    '10000000-0000-0000-0000-000000000003',
    'My Files',
    'Personal root folder for Demo Manager',
    '00000000-0000-0000-0000-000000000003',
    NULL,
    '/10000000-0000-0000-0000-000000000003',
    TRUE,
    FALSE,
    '{}'::jsonb
);

-- Editor root folder
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    metadata
) VALUES (
    '10000000-0000-0000-0000-000000000004',
    'My Files',
    'Personal root folder for Demo Editor',
    '00000000-0000-0000-0000-000000000004',
    NULL,
    '/10000000-0000-0000-0000-000000000004',
    TRUE,
    FALSE,
    '{}'::jsonb
);

-- Viewer root folder
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    metadata
) VALUES (
    '10000000-0000-0000-0000-000000000005',
    'My Files',
    'Personal root folder for Demo Viewer',
    '00000000-0000-0000-0000-000000000005',
    NULL,
    '/10000000-0000-0000-0000-000000000005',
    TRUE,
    FALSE,
    '{}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Shared company folder (owned by super admin, visible to all)
-- -----------------------------------------------------------------------------
INSERT INTO folders (
    id,
    name,
    description,
    owner_id,
    parent_id,
    path,
    is_root,
    is_shared,
    color,
    icon,
    metadata
) VALUES (
    '20000000-0000-0000-0000-000000000001',
    'Company Shared',
    'Organisation-wide shared documents accessible to all team members',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    '/20000000-0000-0000-0000-000000000001',
    FALSE,
    TRUE,
    '#4F46E5',
    'building-office',
    '{"pinned": true}'::jsonb
);

-- -----------------------------------------------------------------------------
-- Grant read permission on the shared folder to all seed users
-- -----------------------------------------------------------------------------
INSERT INTO permissions (resource_id, resource_type, user_id, action, granted_by_id)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000002', 'read',     '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000002', 'download', '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000003', 'read',     '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000003', 'download', '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000003', 'write',    '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000004', 'read',     '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000004', 'download', '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000004', 'write',    '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000005', 'read',     '00000000-0000-0000-0000-000000000001'),
    ('20000000-0000-0000-0000-000000000001', 'folder', '00000000-0000-0000-0000-000000000005', 'download', '00000000-0000-0000-0000-000000000001');

-- -----------------------------------------------------------------------------
-- Audit log — record the seed bootstrap action for traceability
-- -----------------------------------------------------------------------------
INSERT INTO audit_logs (
    user_id,
    action,
    resource_type,
    resource_name,
    ip_address,
    user_agent,
    details,
    status
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'system.seed',
    'system',
    'database_seed',
    '127.0.0.1',
    'migration-runner/1.0',
    '{"migration": "002_seed", "users_created": 5, "folders_created": 6}'::jsonb,
    'success'
);

-- -----------------------------------------------------------------------------
-- Welcome notifications for each seed user
-- -----------------------------------------------------------------------------
INSERT INTO notifications (user_id, type, title, message, metadata)
VALUES
    (
        '00000000-0000-0000-0000-000000000001',
        'system.alert',
        'Welcome to File Management Service',
        'Your super admin account has been set up. Please change your password immediately.',
        '{"priority": "high", "category": "security"}'::jsonb
    ),
    (
        '00000000-0000-0000-0000-000000000002',
        'system.alert',
        'Welcome to File Management Service',
        'Your admin account is ready. Review the platform settings to customise your deployment.',
        '{"priority": "medium", "category": "onboarding"}'::jsonb
    ),
    (
        '00000000-0000-0000-0000-000000000003',
        'permission.granted',
        'Shared Folder Access Granted',
        'You have been granted access to the "Company Shared" folder.',
        '{"resource_type": "folder", "folder_name": "Company Shared"}'::jsonb
    ),
    (
        '00000000-0000-0000-0000-000000000004',
        'permission.granted',
        'Shared Folder Access Granted',
        'You have been granted access to the "Company Shared" folder.',
        '{"resource_type": "folder", "folder_name": "Company Shared"}'::jsonb
    ),
    (
        '00000000-0000-0000-0000-000000000005',
        'permission.granted',
        'Shared Folder Access Granted',
        'You have been granted read-only access to the "Company Shared" folder.',
        '{"resource_type": "folder", "folder_name": "Company Shared"}'::jsonb
    );

COMMIT;

-- =============================================================================
-- Role reference (not a table — kept here for developer reference)
-- =============================================================================
-- super_admin  Full platform control: user management, system settings,
--              unlimited storage, can impersonate any user.
--
-- admin        Manage users (except super_admin), configure platform settings,
--              50 GB default storage quota.
--
-- manager      Create/manage team workspaces, approve share requests,
--              grant permissions up to their own permission level.
--              10 GB default storage quota.
--
-- editor       Upload, edit, delete own files; share files with explicit
--              permission grants; cannot change platform settings.
--              10 GB default storage quota.
--
-- viewer       Read and download files for which they have been explicitly
--              granted permission; cannot upload or modify any resource.
--              5 GB default storage quota.
-- =============================================================================
