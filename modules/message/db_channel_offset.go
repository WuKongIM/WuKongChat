package message

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type channelOffsetDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newChannelOffsetDB(ctx *config.Context) *channelOffsetDB {
	return &channelOffsetDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (c *channelOffsetDB) insertOrUpdate(m *channelOffsetModel) error {
	sq := fmt.Sprintf("INSERT INTO %s (uid,channel_id,channel_type,message_seq) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE message_seq=IF(message_seq<VALUES(message_seq),VALUES(message_seq),message_seq)", "channel_offset")
	_, err := c.session.InsertBySql(sq, m.UID, m.ChannelID, m.ChannelType, m.MessageSeq).Exec()
	return err
}

func (c *channelOffsetDB) queryWithUIDAndChannel(uid string, channelID string, channelType uint8) (*channelOffsetModel, error) {
	var m *channelOffsetModel
	_, err := c.session.Select("*").From("channel_offset").Where("(uid=? or uid='') and channel_id=? and channel_type=?", uid, channelID, channelType).OrderDesc("message_seq").Limit(1).Load(&m)
	return m, err
}

func (c *channelOffsetDB) queryWithUIDAndChannelIDs(uid string, channelIDs []string) ([]*channelOffsetModel, error) {
	var models []*channelOffsetModel
	_, err := c.session.Select("channel_id,channel_type,max(message_seq) message_seq").From("channel_offset").Where("(uid=? or uid='') and channel_id in ?", uid, channelIDs).GroupBy("channel_id", "channel_type").Load(&models)
	return models, err
}

type channelOffsetModel struct {
	UID         string
	ChannelID   string
	ChannelType uint8
	MessageSeq  uint32
	db.BaseModel
}
