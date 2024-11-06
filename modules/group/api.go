package group

import (
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"

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
	}
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
		err = g.db.insert(&GroupModel{GroupNo: groupNo, Name: "群" + groupNo})
		if err != nil {
			g.Error("创建群失败", zap.Error(err))
			c.ResponseError(errors.New("创建群失败"))
			return
		}
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
