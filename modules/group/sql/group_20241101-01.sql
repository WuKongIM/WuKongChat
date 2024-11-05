-- +migrate Up


-- 群组
create table `group`
(
  id         integer     not null primary key AUTO_INCREMENT,
  group_no   VARCHAR(40) not null default '',                             -- 群唯一编号
  name       VARCHAR(40) not null default '',                             -- 群名字
  creator    VARCHAR(40) not null default '',                             -- 创建者
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX group_groupNo on `group` (group_no);
CREATE INDEX group_creator on `group` (creator);

-- -- +migrate StatementBegin
-- CREATE TRIGGER group_updated_at
--   BEFORE UPDATE
--   ON `group` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd