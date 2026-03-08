CREATE TABLE IF NOT EXISTS watchlist (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  user_id bigint(20) NOT NULL DEFAULT 0,
  currency_pair varchar(16) NOT NULL DEFAULT '',
  create_time timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_user_pair (user_id, currency_pair),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SHOW TABLES LIKE 'watchlist';
