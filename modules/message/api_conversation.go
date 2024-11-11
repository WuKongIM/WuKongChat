package message

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/WuKongIM/WuKongIMBusinessExtra/modules/base"
	"go.uber.org/zap"
)

// Conversation 最近会话
type Conversation struct {
	ctx *config.Context
	log.Log
	messageExtraDB                  *messageExtraDB
	messageUserExtraDB              *messageUserExtraDB
	channelOffsetDB                 *channelOffsetDB
	syncConversationResultCacheMap  map[string][]string
	syncConversationVersionMap      map[string]int64
	syncConversationResultCacheLock sync.RWMutex
}

// New New
func NewConversation(ctx *config.Context) *Conversation {
	return &Conversation{
		ctx:                            ctx,
		Log:                            log.NewTLog("Coversation"),
		channelOffsetDB:                newChannelOffsetDB(ctx),
		messageExtraDB:                 newMessageExtraDB(ctx),
		messageUserExtraDB:             newMessageUserExtraDB(ctx),
		syncConversationResultCacheMap: map[string][]string{},
		syncConversationVersionMap:     map[string]int64{},
	}
}

// Route 路由配置
func (co *Conversation) Route(r *wkhttp.WKHttp) {

	conversation := r.Group("/v1/conversation")
	{

		conversation.POST("/sync", co.syncUserConversation) // 离线的最近会话
		conversation.POST("/syncack", co.syncUserConversationAck)
		conversation.PUT("/clearUnread", co.clearUnread) // 清除用户未读消息
	}

}
func (co *Conversation) clearUnread(c *wkhttp.Context) {
	var req struct {
		LoginUID    string `json:"login_uid"`    // 登录用户id
		ChannelID   string `json:"channel_id"`   // 清除某个频道的未读消息
		ChannelType uint8  `json:"channel_type"` // 清除某个频道的未读消息
		Unread      int    `json:"unread"`       // 未读数量 0表示清空所有未读数量
	}
	if err := c.BindJSON(&req); err != nil {
		co.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	err := co.ctx.IMClearConversationUnread(config.ClearConversationUnreadReq{
		UID:         req.LoginUID,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		Unread:      req.Unread,
		MessageSeq:  0,
	})
	if err != nil {
		c.ResponseError(err)
		return
	}
	// 发送清空红点的命令
	err = co.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   req.LoginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDConversationUnreadClear,
		Param: map[string]interface{}{
			"channel_id":   req.ChannelID,
			"channel_type": req.ChannelType,
			"unread":       req.Unread,
		},
	})
	if err != nil {
		co.Error("命令发送失败！", zap.String("cmd", common.CMDConversationUnreadClear))
		c.ResponseError(errors.New("命令发送失败！"))
		return
	}
}

// 获取离线的最近会话
func (co *Conversation) syncUserConversation(c *wkhttp.Context) {
	var req struct {
		LoginUID    string `json:"login_uid"`     // 登录用户id
		Version     int64  `json:"version"`       // 当前客户端的会话最大版本号(客户端最新会话的时间戳)
		LastMsgSeqs string `json:"last_msg_seqs"` // 客户端所有会话的最后一条消息序列号 格式： channelID:channelType:last_msg_seq|channelID:channelType:last_msg_seq
		MsgCount    int64  `json:"msg_count"`     // 每个会话消息数量
		DeviceUUID  string `json:"device_uuid"`   // 设备uuid
	}
	if err := c.BindJSON(&req); err != nil {
		co.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	version := req.Version
	loginUID := req.LoginUID
	lastMsgSeqs := req.LastMsgSeqs
	if !co.ctx.GetConfig().MessageSaveAcrossDevice {
		/**
		1.获取设备的最大version 作为同步version
		2. 如果设备最大version不存在 则把用户最大的version 作为设备version
		**/
		cacheVersion, err := co.getDeviceConversationMaxVersion(loginUID, req.DeviceUUID)
		if err != nil {
			co.Error("获取缓存的最近会话版本号失败！", zap.Error(err), zap.String("loginUID", loginUID), zap.String("deviceUUID", req.DeviceUUID))
			c.ResponseError(errors.New("获取缓存的最近会话版本号失败！"))
			return
		}
		if cacheVersion == 0 {
			userMaxVersion, err := co.getUserConversationMaxVersion(loginUID)
			if err != nil {
				co.Error("获取用户最近会很最大版本失败！", zap.Error(err))
				c.ResponseError(errors.New("获取用户最近会很最大版本失败！"))
				return
			}
			if userMaxVersion > 0 {
				err = co.setDeviceConversationMaxVersion(loginUID, req.DeviceUUID, userMaxVersion)
				if err != nil {
					co.Error("设置设备最近会话最大版本号失败！", zap.Error(err))
					c.ResponseError(errors.New("设置设备最近会话最大版本号失败！"))
					return
				}
			}
			cacheVersion = userMaxVersion
		}
		if cacheVersion > version {
			version = cacheVersion
		}
	}

	resp, err := network.Post(base.APIURL+"/conversation/sync", []byte(util.ToJson(map[string]interface{}{
		"uid":           req.LoginUID,
		"version":       version,
		"last_msg_seqs": lastMsgSeqs,
		"msg_count":     req.MsgCount,
		"larges":        nil,
	})), nil)
	if err != nil {
		co.Error("获取IM离线最近会话失败！", zap.Error(err))
		c.ResponseError(errors.New("获取IM离线最近会话失败！"))
		return
	}
	err = base.HandlerIMError(resp)
	if err != nil {
		c.ResponseError(err)
	}
	var conversations []*config.SyncUserConversationResp
	err = util.ReadJsonByByte([]byte(resp.Body), &conversations)
	if err != nil {
		c.ResponseError(errors.New("解析IM离线最近会话失败！"))
		return
	}
	channelIDs := make([]string, 0, len(conversations))
	if len(conversations) > 0 {
		for _, conversation := range conversations {
			if len(conversation.Recents) == 0 {
				continue
			}

			channelIDs = append(channelIDs, conversation.ChannelID)
		}
	}

	// ---------- 用户频道消息偏移  ----------
	channelOffsetModelMap := map[string]*channelOffsetModel{}
	if len(channelIDs) > 0 {
		channelOffsetModels, err := co.channelOffsetDB.queryWithUIDAndChannelIDs(loginUID, channelIDs)
		if err != nil {
			co.Error("查询用户频道偏移量失败！", zap.Error(err))
			c.ResponseError(errors.New("查询用户频道偏移量失败！"))
			return
		}
		if len(channelOffsetModels) > 0 {
			for _, channelOffsetM := range channelOffsetModels {
				channelOffsetModelMap[fmt.Sprintf("%s-%d", channelOffsetM.ChannelID, channelOffsetM.ChannelType)] = channelOffsetM
			}
		}
	}

	syncUserConversationResps := make([]*SyncUserConversationResp, 0, len(conversations))
	userKey := loginUID
	if len(conversations) > 0 {
		for _, conversation := range conversations {
			channelKey := fmt.Sprintf("%s-%d", conversation.ChannelID, conversation.ChannelType)
			channelOffsetM := channelOffsetModelMap[channelKey]
			// channelSetting := channelSettingMap[channelKey]
			channelOffsetMsgSeq := uint32(0)
			if channelOffsetM != nil {
				channelOffsetMsgSeq = channelOffsetM.MessageSeq
			}

			syncUserConversationResp := newSyncUserConversationResp(conversation, loginUID, co.messageExtraDB, co.messageUserExtraDB, channelOffsetMsgSeq)
			if len(syncUserConversationResp.Recents) > 0 {
				syncUserConversationResps = append(syncUserConversationResps, syncUserConversationResp)
			}
			// if channelSetting != nil {
			// 	syncUserConversationResp.ParentChannelID = channelSetting.ParentChannelID
			// 	syncUserConversationResp.ParentChannelType = channelSetting.ParentChannelType
			// }

			// 缓存频道对应的最新的消息messageSeq
			if !co.ctx.GetConfig().MessageSaveAcrossDevice {

				co.syncConversationResultCacheLock.RLock()
				channelMessageSeqs := co.syncConversationResultCacheMap[userKey]
				co.syncConversationResultCacheLock.RUnlock()
				if channelMessageSeqs == nil {
					channelMessageSeqs = make([]string, 0)
				}
				if len(syncUserConversationResp.Recents) > 0 {
					channelMessageSeqs = append(channelMessageSeqs, co.channelMessageSeqJoin(conversation.ChannelID, conversation.ChannelType, syncUserConversationResp.Recents[0].MessageSeq))
					co.syncConversationResultCacheLock.Lock()
					co.syncConversationResultCacheMap[userKey] = channelMessageSeqs
					co.syncConversationResultCacheLock.Unlock()
				}
			}
		}
	}
	var lastVersion int64 = req.Version
	if len(syncUserConversationResps) > 0 {
		lastVersion = syncUserConversationResps[len(syncUserConversationResps)-1].Version
	}
	co.syncConversationResultCacheLock.Lock()
	cacheVersion := co.syncConversationVersionMap[userKey]
	if cacheVersion < lastVersion {
		co.syncConversationVersionMap[userKey] = lastVersion
	}
	co.syncConversationResultCacheLock.Unlock()

	c.Response(SyncUserConversationRespWrap{
		Conversations: syncUserConversationResps,
		UID:           loginUID,
	})
}

func (co *Conversation) channelMessageSeqJoin(channelID string, channelType uint8, lastMessageSeq uint32) string {
	return fmt.Sprintf("%s:%d:%d", channelID, channelType, lastMessageSeq)
}

func (co *Conversation) syncUserConversationAck(c *wkhttp.Context) {
	var req struct {
		LoginUID   string `json:"login_uid"`   // 登录用户id
		CMDVersion int64  `json:"cmd_version"` // cmd版本
		DeviceUUID string `json:"device_uuid"` // 设备uuid
	}
	if err := c.BindJSON(&req); err != nil {
		co.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if co.ctx.GetConfig().MessageSaveAcrossDevice {
		c.ResponseOK()
		return
	}

	userKey := req.LoginUID

	co.syncConversationResultCacheLock.RLock()
	version := co.syncConversationVersionMap[userKey]
	co.syncConversationResultCacheLock.RUnlock()
	if version > 0 {
		err := co.setUserConversationMaxVersion(req.LoginUID, version)
		if err != nil {
			co.Error("设置设备最近会话最大版本号失败！", zap.Error(err))
			c.ResponseError(errors.New("设置设备最近会话最大版本号失败！"))
			return
		}
	}

	c.ResponseOK()
}

func (co *Conversation) getDeviceConversationMaxVersion(uid string, deviceUUID string) (int64, error) {
	versionStr, err := co.ctx.GetRedisConn().GetString(fmt.Sprintf("deviceMaxVersion:%s-%s", uid, deviceUUID))
	if err != nil {
		return 0, err
	}
	if versionStr == "" {
		return 0, nil
	}
	return strconv.ParseInt(versionStr, 10, 64)
}
func (co *Conversation) setDeviceConversationMaxVersion(uid string, deviceUUID string, version int64) error {
	err := co.ctx.GetRedisConn().Set(fmt.Sprintf("deviceMaxVersion:%s-%s", uid, deviceUUID), fmt.Sprintf("%d", version))
	return err
}

func (co *Conversation) getUserConversationMaxVersion(uid string) (int64, error) {
	versionStr, err := co.ctx.GetRedisConn().GetString(fmt.Sprintf("userMaxVersion:%s", uid))
	if err != nil {
		return 0, err
	}
	if versionStr == "" {
		return 0, nil
	}
	return strconv.ParseInt(versionStr, 10, 64)
}
func (co *Conversation) setUserConversationMaxVersion(uid string, version int64) error {
	err := co.ctx.GetRedisConn().Set(fmt.Sprintf("userMaxVersion:%s", uid), fmt.Sprintf("%d", version))
	return err
}

// ---------- vo ----------

// SyncUserConversationRespWrap SyncUserConversationRespWrap
type SyncUserConversationRespWrap struct {
	UID           string                      `json:"uid"` // 请求者uid
	Conversations []*SyncUserConversationResp `json:"conversations"`
}

type conversationExtraResp struct {
	ChannelID      string `json:"channel_id"`
	ChannelType    uint8  `json:"channel_type"`
	BrowseTo       uint32 `json:"browse_to"`
	KeepMessageSeq uint32 `json:"keep_message_seq"`
	KeepOffsetY    int    `json:"keep_offset_y"`
	Draft          string `json:"draft"` // 草稿
	Version        int64  `json:"version"`
}

// SyncUserConversationResp 最近会话离线返回
type SyncUserConversationResp struct {
	ChannelID       string                 `json:"channel_id"`         // 频道ID
	ChannelType     uint8                  `json:"channel_type"`       // 频道类型
	Unread          int                    `json:"unread,omitempty"`   // 未读消息
	Timestamp       int64                  `json:"timestamp"`          // 最后一次会话时间
	LastMsgSeq      int64                  `json:"last_msg_seq"`       // 最后一条消息seq
	LastClientMsgNo string                 `json:"last_client_msg_no"` // 最后一条客户端消息编号
	OffsetMsgSeq    int64                  `json:"offset_msg_seq"`     // 偏移位的消息seq
	Version         int64                  `json:"version,omitempty"`  // 数据版本
	Recents         []*MsgSyncResp         `json:"recents,omitempty"`  // 最近N条消息
	Extra           *conversationExtraResp `json:"extra,omitempty"`    // 扩展
}

func newSyncUserConversationResp(resp *config.SyncUserConversationResp, loginUID string, messageExtraDB *messageExtraDB, messageUserExtraDB *messageUserExtraDB, channelOffsetMessageSeq uint32) *SyncUserConversationResp {
	recents := make([]*MsgSyncResp, 0, len(resp.Recents))
	lastClientMsgNo := "" // 最新未被删除的消息的clientMsgNo
	if len(resp.Recents) > 0 {
		messageIDs := make([]string, 0, len(resp.Recents))
		for _, message := range resp.Recents {
			messageIDs = append(messageIDs, fmt.Sprintf("%d", message.MessageID))
		}

		// 查询用户个人修改的消息数据
		messageUserExtraModels, err := messageUserExtraDB.queryWithMessageIDsAndUID(messageIDs, loginUID)
		if err != nil {
			log.Error("查询消息编辑字段失败！", zap.Error(err))
		}
		messageUserExtraMap := map[string]*messageUserExtraModel{}
		if len(messageUserExtraModels) > 0 {
			for _, messageUserEditM := range messageUserExtraModels {
				messageUserExtraMap[messageUserEditM.MessageID] = messageUserEditM
			}
		}

		// 消息扩充数据
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

		for _, message := range resp.Recents {

			messageIDStr := strconv.FormatInt(message.MessageID, 10)
			messageExtra := messageExtraMap[messageIDStr]
			messageUserExtra := messageUserExtraMap[messageIDStr]
			msgResp := &MsgSyncResp{}
			msgResp.from(message, loginUID, messageExtra, messageUserExtra, channelOffsetMessageSeq)
			recents = append(recents, msgResp)

			if lastClientMsgNo == "" && msgResp.IsDeleted == 0 {
				lastClientMsgNo = msgResp.ClientMsgNo
			}
		}
	}
	if lastClientMsgNo == "" {
		lastClientMsgNo = resp.LastClientMsgNo
	}

	return &SyncUserConversationResp{
		ChannelID:       resp.ChannelID,
		ChannelType:     resp.ChannelType,
		Unread:          resp.Unread,
		Timestamp:       resp.Timestamp,
		LastMsgSeq:      resp.LastMsgSeq,
		LastClientMsgNo: lastClientMsgNo,
		OffsetMsgSeq:    resp.OffsetMsgSeq,
		Version:         resp.Version,
		Recents:         recents,
	}
}
