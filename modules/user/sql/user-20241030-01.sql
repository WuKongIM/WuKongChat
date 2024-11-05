
-- +migrate Up

-- 用户表
create table `user`
(
  id         integer      not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)  not null default '',                             -- 用户唯一ID
  name       VARCHAR(100) not null default '',                             -- 用户的名字
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid on `user` (uid);

-- -- +migrate StatementBegin
-- CREATE TRIGGER user_updated_at
--   BEFORE UPDATE
--   ON `user` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd