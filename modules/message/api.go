package message

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Message 消息相关API
type Message struct {
	ctx *config.Context
	log.Log
	messageExtraDB     *messageExtraDB
	messageUserExtraDB *messageUserExtraDB
}

// New New
func New(ctx *config.Context) *Message {
	m := &Message{
		ctx:                ctx,
		Log:                log.NewTLog("Message"),
		messageExtraDB:     newMessageExtraDB(ctx),
		messageUserExtraDB: newMessageUserExtraDB(ctx),
	}
	return m
}

// Route 路由配置
func (m *Message) Route(r *wkhttp.WKHttp) {
	message := r.Group("/v1/message")
	{
		message.DELETE("", m.delete)                        // 删除消息
		message.POST("/revoke", m.revoke)                   // 撤回消息
		message.POST("/channel/sync", m.syncChannelMessage) // 同步频道消息
		message.POST("/extra/sync", m.syncMessageExtra)     // 同步消息扩展
	}

}

// 同步扩展消息数据
func (m *Message) syncMessageExtra(c *wkhttp.Context) {
	var req struct {
		LoginUID     string `json:"login_uid"`
		ChannelID    string `json:"channel_id"`
		ChannelType  uint8  `json:"channel_type"`
		ExtraVersion int64  `json:"extra_version"`
		Source       string `json:"source"` // 操作源
		Limit        int    `json:"limit"`  // 数据限制
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(req.LoginUID, req.ChannelID)
	}
	cacheExtraVersion, err := m.getMessageExtraVersion(req.LoginUID, req.Source, fakeChannelID, req.ChannelType)
	if err != nil {
		c.ResponseErrorf("从缓存中获取消息扩展版本失败！", err)
		return
	}
	extraVersion := req.ExtraVersion
	if cacheExtraVersion >= extraVersion {
		extraVersion = cacheExtraVersion
	} else {
		err = m.setMessageExtraVersion(req.LoginUID, fakeChannelID, req.ChannelType, req.Source, extraVersion)
		if err != nil {
			c.ResponseErrorf("缓存最大的消息扩展版本失败！", err)
			return
		}

	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	if strings.TrimSpace(req.ChannelID) == "" {
		c.ResponseError(errors.New("频道ID不能为空！"))
		return
	}
	extraModels, err := m.messageExtraDB.sync(extraVersion, fakeChannelID, req.ChannelType, uint64(limit))
	if err != nil {
		c.ResponseErrorf("同步消息扩展数据失败！", err)
		return
	}
	resps := make([]*messageExtraResp, 0, len(extraModels))
	if len(extraModels) > 0 {
		for _, extraModel := range extraModels {
			resps = append(resps, newMessageExtraResp(extraModel))
		}
	}
	c.Response(resps)
}

// 同步频道消息
func (m *Message) syncChannelMessage(c *wkhttp.Context) {
	var req *config.SyncChannelMessageReq
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	resp, err := network.Post(base.APIURL+"/channel/messagesync", []byte(util.ToJson(req)), nil)
	if err != nil {
		m.Error("同步频道消息错误", zap.Error(err))
		c.ResponseError(errors.New("同步频道消息错误"))
		return
	}
	err = base.HandlerIMError(resp)
	if err != nil {
		c.ResponseError(err)
		return
	}
	var syncChannelMessageResp *config.SyncChannelMessageResp
	err = util.ReadJsonByByte([]byte(resp.Body), &syncChannelMessageResp)
	if err != nil {
		m.Error("同步频道消息解析错误", zap.Error(err))
		c.ResponseError(errors.New("同步频道消息解析错误"))
		return
	}

	fmt.Println("resp----messages-->", len(syncChannelMessageResp.Messages))

	c.Response(newSyncChannelMessageResp(syncChannelMessageResp, req.LoginUID, m.messageExtraDB, m.messageUserExtraDB))
}

// 删除消息
func (m *Message) delete(c *wkhttp.Context) {
	var req *deleteReq
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if err := req.check(); err != nil {
		c.ResponseError(err)
		return
	}

	err := m.messageUserExtraDB.insertOrUpdateDeleted(&messageUserExtraModel{
		UID:              req.LoginUID,
		MessageID:        req.MessageID,
		MessageSeq:       req.MessageSeq,
		ChannelID:        req.ChannelID,
		ChannelType:      req.ChannelType,
		MessageIsDeleted: 1,
	})
	if err != nil {
		m.Error("删除消息失败！", zap.Error(err))
		c.ResponseError(errors.New("删除消息失败！"))
		return
	}
	c.ResponseOK()
}

// 撤回消息
func (m *Message) revoke(c *wkhttp.Context) {
	var req *revokeReq
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	if err := req.check(); err != nil {
		c.ResponseError(err)
		return
	}
	fakeChannelID := req.ChannelID
	if uint8(req.ChannelType) == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(req.ChannelID, req.LoginUID)
	}

	// 如果撤回的是艾特消息需要删除对应的提醒记录
	// 具体可查看唐僧叨叨实现逻辑
	// https://github.com/TangSengDaoDao/TangSengDaoDaoServer/blob/main/modules/message/api.go

	// 这里需要查询clientMsgNo下的所有消息（存在重试消息，clientMsgNo相同，messageID不同），然后遍历进行撤回标记。
	// 演示程序并没有实现这一步，具体可查看唐僧叨叨实现逻辑。
	// https://github.com/TangSengDaoDao/TangSengDaoDaoServer/blob/main/modules/message/api.go revoke 方法
	messageExtr, err := m.messageExtraDB.queryWithMessageID(req.MessageID)
	if err != nil {
		m.Error("查询消息扩展错误", zap.Error(err))
		c.ResponseError(errors.New("查询消息扩展错误"))
		return
	}
	version := time.Now().Unix()
	if messageExtr == nil {
		err = m.messageExtraDB.insert(&messageExtraModel{
			MessageID:   req.MessageID,
			MessageSeq:  0,
			FromUID:     "",
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			ReadedCount: 0,
			Version:     version,
			Revoke:      1,
			Revoker:     req.LoginUID,
		})
		if err != nil {
			m.Error("新增消息扩展数据失败！", zap.Error(err), zap.String("messageID", req.MessageID), zap.String("channelID", fakeChannelID))
			return
		}
	} else {
		messageExtr.Revoke = 1
		messageExtr.Revoker = req.LoginUID
		messageExtr.Version = version
		err = m.messageExtraDB.update(messageExtr)
		if err != nil {
			m.Error("更新消息扩展数据失败！", zap.Error(err), zap.String("messageID", req.MessageID), zap.String("channelID", fakeChannelID))
			return
		}
	}
	messageIDI, _ := strconv.ParseInt(req.MessageID, 10, 64)
	// 发给指定频道
	err = m.ctx.SendRevoke(&config.MsgRevokeReq{
		Operator:     req.LoginUID,
		OperatorName: req.LoginUID,
		FromUID:      req.LoginUID,
		ChannelID:    req.ChannelID,
		ChannelType:  req.ChannelType,
		MessageID:    messageIDI,
	})
	if err != nil {
		m.Error("发送撤回消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送撤回消息失败！"))
		return
	}
	c.ResponseOK()

}

func (m *Message) getMessageExtraVersion(uid, source, channelID string, channelType uint8) (int64, error) {
	versionStr, err := m.ctx.GetRedisConn().Hget(fmt.Sprintf("messageExtraVersion:%s%s", uid, source), fmt.Sprintf("%s-%d", channelID, channelType))
	if err != nil {
		return 0, err
	}
	if versionStr == "" {
		return 0, nil
	}
	version, _ := strconv.ParseInt(versionStr, 10, 64)
	return version, nil

}

func (m *Message) setMessageExtraVersion(uid, channelID string, channelType uint8, source string, messageExtraVersion int64) error {
	err := m.ctx.GetRedisConn().Hset(fmt.Sprintf("messageExtraVersion:%s%s", uid, source), fmt.Sprintf("%s-%d", channelID, channelType), fmt.Sprintf("%d", messageExtraVersion))
	if err != nil {
		return err
	}
	return nil
}

// ---------- vo ----------

type syncChannelMessageResp struct {
	StartMessageSeq uint32          `json:"start_message_seq"` // 开始序列号
	EndMessageSeq   uint32          `json:"end_message_seq"`   // 结束序列号
	PullMode        config.PullMode `json:"pull_mode"`         // 拉取模式
	More            int             `json:"more"`              // 是否还有更多 1.是 0.否
	Messages        []*MsgSyncResp  `json:"messages"`          // 消息数据
}

func newSyncChannelMessageResp(resp *config.SyncChannelMessageResp, loginUID string, messageExtraDB *messageExtraDB, messageUserExtraDB *messageUserExtraDB) *syncChannelMessageResp {
	messages := make([]*MsgSyncResp, 0, len(resp.Messages))
	if len(resp.Messages) > 0 {
		messageIDs := make([]string, 0, len(resp.Messages))
		for _, message := range resp.Messages {
			messageIDs = append(messageIDs, fmt.Sprintf("%d", message.MessageID))
		}

		// 消息全局扩张
		messageExtras, err := messageExtraDB.queryWithMessageIDs(messageIDs)
		if err != nil {
			log.Error("查询消息扩展字段失败！", zap.Error(err))
		}
		messageExtraMap := map[string]*messageExtraModel{}
		if len(messageExtras) > 0 {
			for _, messageExtra := range messageExtras {
				messageExtraMap[messageExtra.MessageID] = messageExtra
			}
		}

		// 消息用户扩张
		messageUserExtras, err := messageUserExtraDB.queryWithMessageIDsAndUID(messageIDs, loginUID)
		if err != nil {
			log.Error("查询用户消息扩展字段失败！", zap.Error(err))
		}
		messageUserExtraMap := map[string]*messageUserExtraModel{}
		if len(messageUserExtras) > 0 {
			for _, messageUserExtraM := range messageUserExtras {
				messageUserExtraMap[messageUserExtraM.MessageID] = messageUserExtraM
			}
		}

		// 设备偏移
		for _, message := range resp.Messages {
			messageIDStr := strconv.FormatInt(message.MessageID, 10)
			messageExtra := messageExtraMap[messageIDStr]
			messageUserExtra := messageUserExtraMap[messageIDStr]
			msgResp := &MsgSyncResp{}
			msgResp.from(message, loginUID, messageExtra, messageUserExtra)
			messages = append(messages, msgResp)
		}
	}
	return &syncChannelMessageResp{
		StartMessageSeq: resp.StartMessageSeq,
		EndMessageSeq:   resp.EndMessageSeq,
		PullMode:        resp.PullMode,
		Messages:        messages,
	}
}

// 消息头
type messageHeader struct {
	NoPersist int `json:"no_persist"` // 是否不持久化
	RedDot    int `json:"red_dot"`    // 是否显示红点
	SyncOnce  int `json:"sync_once"`  // 此消息只被同步或被消费一次
}

// MgSyncResp 消息同步请求
type MsgSyncResp struct {
	Header        messageHeader          `json:"header"`                    // 消息头部
	Setting       uint8                  `json:"setting"`                   // 设置
	MessageID     int64                  `json:"message_id"`                // 服务端的消息ID(全局唯一)
	MessageIDStr  string                 `json:"message_idstr"`             // 服务端的消息ID(全局唯一)字符串形式
	MessageSeq    uint32                 `json:"message_seq"`               // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo   string                 `json:"client_msg_no"`             // 客户端消息唯一编号
	StreamNo      string                 `json:"stream_no,omitempty"`       // 流编号
	FromUID       string                 `json:"from_uid"`                  // 发送者UID
	ToUID         string                 `json:"to_uid,omitempty"`          // 接受者uid
	ChannelID     string                 `json:"channel_id"`                // 频道ID
	ChannelType   uint8                  `json:"channel_type"`              // 频道类型
	Expire        uint32                 `json:"expire,omitempty"`          // expire
	Timestamp     int32                  `json:"timestamp"`                 // 服务器消息时间戳(10位，到秒)
	Payload       map[string]interface{} `json:"payload"`                   // 消息内容
	SignalPayload string                 `json:"signal_payload"`            // signal 加密后的payload base64编码,TODO: 这里为了兼容没加密的版本，所以新用SignalPayload字段
	ReplyCount    int                    `json:"reply_count,omitempty"`     // 回复集合
	ReplyCountSeq string                 `json:"reply_count_seq,omitempty"` // 回复数量seq
	ReplySeq      string                 `json:"reply_seq,omitempty"`       // 回复seq
	IsDeleted     int                    `json:"is_deleted"`                // 是否已删除
	VoiceStatus   int                    `json:"voice_status,omitempty"`    // 语音状态 0.未读 1.已读
	Streams       []*streamItemResp      `json:"streams,omitempty"`         // 流数据
	// ---------- 旧字段 这些字段都放到MessageExtra对象里了 ----------
	Readed       int    `json:"readed"`                 // 是否已读（针对于自己）
	Revoke       int    `json:"revoke,omitempty"`       // 是否撤回
	Revoker      string `json:"revoker,omitempty"`      // 消息撤回者
	ReadedCount  int    `json:"readed_count,omitempty"` // 已读数量
	UnreadCount  int    `json:"unread_count,omitempty"` // 未读数量
	ExtraVersion int64  `json:"extra_version"`          // 扩展数据版本号

	// 消息扩展字段
	MessageExtra *messageExtraResp `json:"message_extra,omitempty"` // 消息扩展

}

func (m *MsgSyncResp) from(msgResp *config.MessageResp, loginUID string, messageExtraM *messageExtraModel, messageUserExtraM *messageUserExtraModel) {
	m.Header.NoPersist = msgResp.Header.NoPersist
	m.Header.RedDot = msgResp.Header.RedDot
	m.Header.SyncOnce = msgResp.Header.SyncOnce
	m.Setting = msgResp.Setting
	m.MessageID = msgResp.MessageID
	m.MessageIDStr = strconv.FormatInt(msgResp.MessageID, 10)
	m.MessageSeq = msgResp.MessageSeq
	m.ClientMsgNo = msgResp.ClientMsgNo
	m.StreamNo = msgResp.StreamNo
	m.FromUID = msgResp.FromUID
	m.ToUID = msgResp.ToUID
	m.ChannelID = msgResp.ChannelID
	m.ChannelType = msgResp.ChannelType
	m.Expire = msgResp.Expire
	m.Timestamp = msgResp.Timestamp
	if messageExtraM != nil {
		// TODO: 后续这些字段可以废除了 都放MessageExtra对象里了
		m.IsDeleted = messageExtraM.IsDeleted
		m.Revoke = messageExtraM.Revoke
		m.Revoker = messageExtraM.Revoker
		m.ReadedCount = messageExtraM.ReadedCount
		m.ExtraVersion = messageExtraM.Version

		m.MessageExtra = newMessageExtraResp(messageExtraM)
	}

	setting := config.SettingFromUint8(msgResp.Setting)
	var payloadMap map[string]interface{}
	if setting.Signal {
		m.SignalPayload = base64.StdEncoding.EncodeToString(msgResp.Payload)
		payloadMap = map[string]interface{}{
			"type": common.SignalError.Int(),
		}
	} else {
		err := util.ReadJsonByByte(msgResp.Payload, &payloadMap)
		if err != nil {
			log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msgResp.Payload)))
		}
		if len(payloadMap) > 0 {
			visibles := payloadMap["visibles"]
			if visibles != nil {
				visiblesArray := visibles.([]interface{})
				if len(visiblesArray) > 0 {
					m.IsDeleted = 1
					for _, limitUID := range visiblesArray {
						if limitUID == loginUID {
							m.IsDeleted = 0
						}
					}
				}
			}
		} else {
			payloadMap = map[string]interface{}{
				"type": common.ContentError.Int(),
			}
		}
	}

	if messageUserExtraM != nil {
		if m.IsDeleted == 0 {
			m.IsDeleted = messageUserExtraM.MessageIsDeleted
		}
		m.VoiceStatus = messageUserExtraM.VoiceReaded
	}

	if msgResp.Expire > 0 {
		if time.Now().Unix()-int64(msgResp.Expire) >= int64(msgResp.Timestamp) {
			m.IsDeleted = 1
		}
	}
	// if channelOffsetMessageSeq != 0 && msgResp.MessageSeq <= channelOffsetMessageSeq {
	// 	m.IsDeleted = 1
	// }
	m.Payload = payloadMap

	// msgReactionList := make([]*reactionSimpleResp, 0, len(reactionModels))
	// if len(reactionModels) > 0 {
	// 	for _, reaction := range reactionModels {
	// 		msgReactionList = append(msgReactionList, &reactionSimpleResp{
	// 			UID:       reaction.UID,
	// 			Name:      reaction.Name,
	// 			Seq:       reaction.Seq,
	// 			IsDeleted: reaction.IsDeleted,
	// 			Emoji:     reaction.Emoji,
	// 			CreatedAt: reaction.CreatedAt.String(),
	// 		})
	// 	}
	// }
	// m.Reactions = msgReactionList

	if len(msgResp.Streams) > 0 {
		streams := make([]*streamItemResp, 0, len(msgResp.Streams))
		for _, streamItem := range msgResp.Streams {
			streams = append(streams, newStreamItemResp(streamItem))
		}
		m.Streams = streams
	}

}

type streamItemResp struct {
	StreamSeq   uint32         `json:"stream_seq"`    // 流序号
	ClientMsgNo string         `json:"client_msg_no"` // 客户端消息唯一编号
	Blob        map[string]any `json:"blob"`          // 消息内容
}

func newStreamItemResp(streamItem *config.StreamItemResp) *streamItemResp {
	var blobMap map[string]any
	err := util.ReadJsonByByte(streamItem.Blob, &blobMap)
	if err != nil {
		log.Warn("blob不是json格式！", zap.Error(err), zap.String("blob", string(streamItem.Blob)))
	}
	return &streamItemResp{
		ClientMsgNo: streamItem.ClientMsgNo,
		StreamSeq:   streamItem.StreamSeq,
		Blob:        blobMap,
	}
}

type messageExtraResp struct {
	MessageID       int64                  `json:"message_id"`
	MessageIDStr    string                 `json:"message_id_str"`
	Revoke          int                    `json:"revoke,omitempty"`
	Revoker         string                 `json:"revoker,omitempty"`
	VoiceStatus     int                    `json:"voice_status,omitempty"`
	Readed          int                    `json:"readed,omitempty"`            // 是否已读（针对于自己）
	ReadedCount     int                    `json:"readed_count,omitempty"`      // 已读数量
	ReadedAt        int64                  `json:"readed_at,omitempty"`         // 已读时间
	IsMutualDeleted int                    `json:"is_mutual_deleted,omitempty"` // 双向删除
	IsPinned        int                    `json:"is_pinned,omitempty"`         // 是否置顶
	ContentEdit     map[string]interface{} `json:"content_edit,omitempty"`      // 编辑后的正文
	EditedAt        int                    `json:"edited_at,omitempty"`         // 编辑时间 例如 12:23
	ExtraVersion    int64                  `json:"extra_version"`               // 数据版本
}

type deleteReq struct {
	LoginUID    string `json:"login_uid"`
	MessageID   string `json:"message_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	MessageSeq  uint32 `json:"message_seq"`
}
type revokeReq struct {
	LoginUID    string `json:"login_uid"`
	MessageID   string `json:"message_id"`
	ClientMsgNo string `json:"client_msg_no"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}

func (r *revokeReq) check() error {
	if strings.TrimSpace(r.MessageID) == "" {
		return errors.New("消息ID不能为空！")
	}
	if strings.TrimSpace(r.ClientMsgNo) == "" {
		return errors.New("客户端消息唯一编号不能为空！")
	}
	if strings.TrimSpace(r.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if strings.TrimSpace(r.LoginUID) == "" {
		return errors.New("uid不能为空")
	}
	return nil
}
func (d *deleteReq) check() error {
	if strings.TrimSpace(d.MessageID) == "" {
		return errors.New("消息ID不能为空！")
	}
	if strings.TrimSpace(d.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if strings.TrimSpace(d.LoginUID) == "" {
		return errors.New("uid不能为空")
	}
	if d.ChannelType == 0 {
		return errors.New("频道类型不能为空！")
	}
	if d.MessageSeq == 0 {
		return errors.New("消息序号不能为空！")
	}
	return nil
}

func newMessageExtraResp(m *messageExtraModel) *messageExtraResp {

	messageID, _ := strconv.ParseInt(m.MessageID, 10, 64)

	var contentEditMap map[string]interface{}
	if m.ContentEdit.String != "" {
		err := util.ReadJsonByByte([]byte(m.ContentEdit.String), &contentEditMap)
		if err != nil {
			log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(m.ContentEdit.String)))
		}
	}

	var readedAt int64 = 0

	return &messageExtraResp{
		MessageID:       messageID,
		MessageIDStr:    m.MessageID,
		Revoke:          m.Revoke,
		Revoker:         m.Revoker,
		ReadedAt:        readedAt,
		ReadedCount:     m.ReadedCount,
		ContentEdit:     contentEditMap,
		EditedAt:        m.EditedAt,
		IsMutualDeleted: m.IsDeleted,
		ExtraVersion:    m.Version,
	}
}
