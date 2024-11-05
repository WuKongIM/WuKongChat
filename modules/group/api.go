package group

import (
	"errors"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
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

	group := r.Group("/v1/group")
	{
		group.POST("/create", g.create) // 创建群
	}
	groups := r.Group("/v1/groups")
	{
		groups.GET("/:group_no", g.groupGet) // 群详情
	}
}

// create 创建群
func (g *Group) create(c *wkhttp.Context) {
	var req CreateReq
	if err := c.Bind(&req); err != nil {
		g.Error("参数错误", zap.Error(err))
		c.ResponseError(errors.New("参数错误"))
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
		if err := g.db.insert(&GroupModel{GroupNo: req.GroupNo, Name: name}); err != nil {
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
		c.ResponseError(errors.New("群不存在"))
		return
	}

	c.Response(&GroupResp{
		GroupNo: model.GroupNo,
		Name:    model.Name,
	})
}

type GroupResp struct {
	GroupNo string `json:"group_no"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
}

type CreateReq struct {
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
