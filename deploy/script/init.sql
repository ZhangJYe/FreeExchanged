CREATE DATABASE IF NOT EXISTS freeexchanged;
USE freeexchanged;

CREATE TABLE IF NOT EXISTS `user` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `username` varchar(255) NOT NULL DEFAULT '',
    `password` varchar(255) NOT NULL DEFAULT '',
    `mobile` varchar(20) NOT NULL DEFAULT '',
    `nickname` varchar(255) NOT NULL DEFAULT '',
    `avatar` varchar(255) NOT NULL DEFAULT '',
    `info` varchar(255) NOT NULL DEFAULT '',
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_mobile_unique` (`mobile`),
    UNIQUE KEY `idx_username_unique` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `articles` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `title` varchar(255) NOT NULL DEFAULT '',
    `content` text NOT NULL,
    `author_id` bigint(20) NOT NULL DEFAULT 0,
    `status` bigint(20) NOT NULL DEFAULT 0,
    `like_count` bigint(20) NOT NULL DEFAULT 0,
    `view_count` bigint(20) NOT NULL DEFAULT 0,
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_author_id` (`author_id`),
    KEY `idx_status_create_time` (`status`, `create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `article_outbox_events` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `aggregate_type` varchar(64) NOT NULL DEFAULT '',
    `aggregate_id` bigint(20) NOT NULL DEFAULT 0,
    `event_type` varchar(128) NOT NULL DEFAULT '',
    `topic` varchar(128) NOT NULL DEFAULT '',
    `event_key` varchar(128) NOT NULL DEFAULT '',
    `payload` json NOT NULL,
    `status` tinyint(4) NOT NULL DEFAULT 0,
    `retry_count` int NOT NULL DEFAULT 0,
    `last_error` varchar(1024) NOT NULL DEFAULT '',
    `next_retry_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `locked_by` varchar(128) NOT NULL DEFAULT '',
    `locked_until` timestamp NULL DEFAULT NULL,
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_status_next_retry` (`status`, `next_retry_at`, `id`),
    KEY `idx_status_lock` (`status`, `locked_until`, `id`),
    KEY `idx_aggregate` (`aggregate_type`, `aggregate_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET @stmt = IF(
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'article_outbox_events' AND column_name = 'locked_by') = 0,
    'ALTER TABLE `article_outbox_events` ADD COLUMN `locked_by` varchar(128) NOT NULL DEFAULT ''''',
    'SELECT 1'
);
PREPARE stmt FROM @stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @stmt = IF(
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'article_outbox_events' AND column_name = 'locked_until') = 0,
    'ALTER TABLE `article_outbox_events` ADD COLUMN `locked_until` timestamp NULL DEFAULT NULL',
    'SELECT 1'
);
PREPARE stmt FROM @stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @stmt = IF(
    (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'article_outbox_events' AND index_name = 'idx_status_lock') = 0,
    'ALTER TABLE `article_outbox_events` ADD INDEX `idx_status_lock` (`status`, `locked_until`, `id`)',
    'SELECT 1'
);
PREPARE stmt FROM @stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `interaction_states` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `user_id` bigint(20) NOT NULL DEFAULT 0,
    `article_id` bigint(20) NOT NULL DEFAULT 0,
    `liked` tinyint(1) NOT NULL DEFAULT 0,
    `read_count` bigint(20) NOT NULL DEFAULT 0,
    `last_read_at` timestamp NULL DEFAULT NULL,
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_user_article` (`user_id`, `article_id`),
    KEY `idx_article_liked` (`article_id`, `liked`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `interaction_outbox_events` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `aggregate_type` varchar(64) NOT NULL DEFAULT '',
    `aggregate_id` bigint(20) NOT NULL DEFAULT 0,
    `event_type` varchar(128) NOT NULL DEFAULT '',
    `topic` varchar(128) NOT NULL DEFAULT '',
    `event_key` varchar(128) NOT NULL DEFAULT '',
    `payload` json NOT NULL,
    `status` tinyint(4) NOT NULL DEFAULT 0,
    `retry_count` int NOT NULL DEFAULT 0,
    `last_error` varchar(1024) NOT NULL DEFAULT '',
    `next_retry_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `locked_by` varchar(128) NOT NULL DEFAULT '',
    `locked_until` timestamp NULL DEFAULT NULL,
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_status_next_retry` (`status`, `next_retry_at`, `id`),
    KEY `idx_status_lock` (`status`, `locked_until`, `id`),
    KEY `idx_aggregate` (`aggregate_type`, `aggregate_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `watchlist` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `user_id` bigint(20) NOT NULL DEFAULT 0,
    `currency_pair` varchar(16) NOT NULL DEFAULT '',
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_user_pair` (`user_id`, `currency_pair`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
