-- Jarvis 2.0 User Service Database Initialization
-- This script creates the necessary tables for the user authentication and authorization service
-- based on the GORM models.

-- 1. Permissions Table: Stores individual, granular permissions.
CREATE TABLE IF NOT EXISTS `permissions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime(3) NULL,
    `updated_at` datetime(3) NULL,
    `deleted_at` datetime(3) NULL,
    `name` varchar(255) UNIQUE NOT NULL COMMENT 'Permission name, e.g., users:create',
    `description` varchar(1024) COMMENT 'Detailed description of the permission',
    PRIMARY KEY (`id`),
    INDEX `idx_permissions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. Roles Table: Stores roles that group multiple permissions.
CREATE TABLE IF NOT EXISTS `roles` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime(3) NULL,
    `updated_at` datetime(3) NULL,
    `deleted_at` datetime(3) NULL,
    `name` varchar(255) UNIQUE NOT NULL COMMENT 'Role name, e.g., Admin, Member, VIP',
    `description` varchar(1024) COMMENT 'Detailed description of the role',
    PRIMARY KEY (`id`),
    INDEX `idx_roles_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. Users Table: Stores user account information.
CREATE TABLE IF NOT EXISTS `users` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime(3) NULL,
    `updated_at` datetime(3) NULL,
    `deleted_at` datetime(3) NULL,
    `username` varchar(191) UNIQUE NOT NULL,
    `full_name` varchar(255),
    `email` varchar(191) UNIQUE NOT NULL,
    `password` varchar(255) NULL COMMENT 'Hashed password for email login',
    `avatar_url` longtext,
    `provider` varchar(50) NOT NULL COMMENT 'OAuth provider, e.g., google, github',
    `provider_id` varchar(191) NOT NULL COMMENT 'User ID from the OAuth provider',
    `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'User account status (pending, active, suspended)',
    `last_login_at` datetime(3) NULL,
    `tenant_id` bigint unsigned,
    `settings` json,
    PRIMARY KEY (`id`),
    INDEX `idx_users_deleted_at` (`deleted_at`),
    UNIQUE INDEX `idx_provider_user` (`provider`, `provider_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. Role-Permissions Join Table (Many-to-Many).
CREATE TABLE IF NOT EXISTS `role_permissions` (
    `auth_role_id` bigint unsigned NOT NULL,
    `permission_id` bigint unsigned NOT NULL,
    PRIMARY KEY (`auth_role_id`, `permission_id`),
    CONSTRAINT `fk_role_permissions_auth_role` FOREIGN KEY (`auth_role_id`) REFERENCES `roles`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. User-Roles Join Table (Many-to-Many).
CREATE TABLE IF NOT EXISTS `user_roles` (
    `user_id` bigint unsigned NOT NULL,
    `auth_role_id` bigint unsigned NOT NULL,
    PRIMARY KEY (`user_id`, `auth_role_id`),
    CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_user_roles_auth_role` FOREIGN KEY (`auth_role_id`) REFERENCES `roles`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 6. RAG Folders Table: Stores user-defined folders for categorizing documents.
CREATE TABLE IF NOT EXISTS `rag_folders` (
    `id` int unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime(3) NULL,
    `updated_at` datetime(3) NULL,
    `user_id` varchar(255) NOT NULL,
    `name` varchar(255) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_user_folder` (`user_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;