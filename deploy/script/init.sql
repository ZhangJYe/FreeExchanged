CREATE DATABASE IF NOT EXISTS freeexchanged;
USE freeexchanged;

CREATE TABLE `user` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `username` varchar(255) NOT NULL DEFAULT '' COMMENT '用户名',
    `password` varchar(255) NOT NULL DEFAULT '' COMMENT '加密后的密码',
    `mobile` varchar(20) NOT NULL DEFAULT '' COMMENT '手机号',
    `nickname` varchar(255) NOT NULL DEFAULT '' COMMENT '昵称',
    `avatar` varchar(255) NOT NULL DEFAULT '' COMMENT '头像',
    `info` varchar(255) NOT NULL DEFAULT '' COMMENT '简介',
    `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_mobile_unique` (`mobile`),
    UNIQUE KEY `idx_username_unique` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;