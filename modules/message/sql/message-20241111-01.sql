-- +migrate Up

-- 频道偏移表 （每个用户针对于频道的偏移位置）
CREATE TABLE `channel_offset`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_channel_idx on `channel_offset` (uid,channel_id,channel_type);
