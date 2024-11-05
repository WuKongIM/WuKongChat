-- +migrate Up

-- 消息扩展表
create table `message_extra`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(严格递增)
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  from_uid   VARCHAR(40)      not null default '', -- 发送者uid
  `revoke`      smallint     not null default 0,  -- 是否撤回
  revoker       VARCHAR(40)   not null default '',  -- 是否撤回
  clone_no     VARCHAR(40)   not null default '', -- 未读编号
  -- voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  `version`       bigint          not null default 0, -- 数据版本
  readed_count  integer     not null default 0,  -- 已读数量
  is_deleted  smallint     not null default 0,  -- 是否已删除
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
); 
CREATE  INDEX from_uid_idx on `message_extra` (from_uid);
CREATE  INDEX channel_idx on `message_extra` (channel_id,channel_type);
CREATE UNIQUE INDEX message_id on `message_extra` (message_id);