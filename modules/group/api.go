package group

import (
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/WuKongIM/WuKongIMBusinessExtra/modules/base"
	"go.uber.org/zap"
)

// Group 群组相关API
type Group struct {
	ctx *config.Context
	log.Log
	db *DB
}

// New New
func New(ctx *config.Context) *Group {

	g := &Group{
		ctx: ctx,
		Log: log.NewTLog("Group"),
		db:  NewDB(ctx),
	}
	return g
}

// Route 路由配置
func (g *Group) Route(r *wkhttp.WKHttp) {

	v := r.Group("/v1")
	{
		v.POST("/group/create", g.create)                // 创建群
		v.GET("/groups/:group_no", g.groupGet)           // 群详情
		v.GET("/groups/:group_no/avatar", g.groupAvatar) // 群头像
		v.PUT("/groups/:group_no", g.groupUpdateName)    // 更新群资料
	}
}

// 更新群名称
func (g *Group) groupUpdateName(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	var req groupUpdateNameReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if req.Name == "" {
		c.ResponseError(errors.New("群名称不能为空"))
		return
	}
	if err := g.db.updateName(req.Name, groupNo); err != nil {
		g.Error("更新群名称失败", zap.Error(err))
		c.ResponseError(errors.New("更新群名称失败"))
		return
	}
	// 发送cmd更新群资料
	err := base.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		FromUID:     req.LoginUID,
		CMD:         common.CMDChannelUpdate,
		Param: map[string]interface{}{
			"channel_id":   groupNo,
			"channel_type": common.ChannelTypeGroup.Uint8(),
		},
	})
	if err != nil {
		g.Error("发送更新群资料cmd错误", zap.Error(err))
		c.ResponseError(errors.New("发送更新群资料cmd错误"))
		return
	}
	c.ResponseOK()
}

// groupAvatar 群头像
func (g *Group) groupAvatar(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	avatarID := crc32.ChecksumIEEE([]byte(groupNo)) % uint32(20)
	path := fmt.Sprintf("assets/assets/avatar/g_%d.jpeg", avatarID)
	c.Header("Content-Type", "image/jpeg")
	avatarBytes, err := os.ReadFile(path)
	if err != nil {
		g.Error("头像读取失败！", zap.Error(err))
		c.Writer.WriteHeader(http.StatusNotFound)
		return
	}
	c.Writer.Write(avatarBytes)
}

// create 创建群
func (g *Group) create(c *wkhttp.Context) {
	var req createReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if req.GroupNo == "" {
		c.ResponseError(errors.New("群号不能为空"))
		return
	}
	if req.LoginUID == "" {
		c.ResponseError(errors.New("登录用户ID不能为空"))
		return
	}
	model, err := g.db.query(req.GroupNo)
	if err != nil {
		g.Error("查询群资料错误", zap.Error(err))
		c.ResponseError(errors.New("查询群资料错误"))
		return
	}
	if model == nil {
		name := fmt.Sprintf("群%s", req.GroupNo)
		if err := g.db.insert(&GroupModel{GroupNo: req.GroupNo, Name: name, Creator: req.LoginUID}); err != nil {
			g.Error("创建群失败", zap.Error(err))
			c.ResponseError(errors.New("创建群失败"))
			return
		}
	}
	// 添加频道订阅者
	resp, err := network.Post(base.APIURL+"/channel/subscriber_add", []byte(util.ToJson(&subscriberAddReq{
		ChannelID:   req.GroupNo,
		ChannelType: 2,
		Reset:       0,
		Subscribers: []string{req.LoginUID},
	})), nil)
	if err != nil {
		g.Error("添加订阅错误", zap.Error(err))
		c.ResponseError(errors.New("添加订阅错误"))
		return
	}
	err = base.HandlerIMError(resp)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

// 获取群详情
func (g *Group) groupGet(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	model, err := g.db.query(groupNo)
	if err != nil {
		g.Error("查询群资料错误", zap.Error(err))
		c.ResponseError(errors.New("查询群资料错误"))
		return
	}
	if model == nil {
		name := "群" + groupNo
		err = g.db.insert(&GroupModel{GroupNo: groupNo, Name: name})
		if err != nil {
			g.Error("创建群失败", zap.Error(err))
			c.ResponseError(errors.New("创建群失败"))
			return
		}
		model = &GroupModel{GroupNo: groupNo, Name: name}
	}
	avatar := fmt.Sprintf("groups/%s/avatar", groupNo)
	c.Response(&groupResp{
		GroupNo: model.GroupNo,
		Name:    model.Name,
		Avatar:  avatar,
	})
}

type groupResp struct {
	GroupNo string `json:"group_no"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
}
type groupUpdateNameReq struct {
	LoginUID string `json:"login_uid"`
	Name     string `json:"name"`
}
type createReq struct {
	GroupNo  string `json:"group_no"`
	LoginUID string `json:"login_uid"`
}

// subscriberAddReq 添加订阅请求
type subscriberAddReq struct {
	ChannelID   string   `json:"channel_id"`
	ChannelType uint8    `json:"channel_type"`
	Reset       int      `json:"reset"` // 是否重置订阅者 （0.不重置 1.重置），选择重置，将删除原来的所有成员
	Subscribers []string `json:"subscribers"`
}
