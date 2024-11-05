package webhook

import (
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

// 数据源
func (w *Webhook) datasource(c *wkhttp.Context) {
	var cmdReq struct {
		CMD  string                 `json:"cmd"`
		Data map[string]interface{} `json:"data"`
	}
	if err := c.BindJSON(&cmdReq); err != nil {
		w.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(cmdReq.CMD) == "" {
		c.ResponseError(errors.New("cmd不能为空！"))
		return
	}
	w.Debug("请求数据源", zap.Any("cmd", cmdReq))
	var result interface{}
	var err error
	switch cmdReq.CMD {
	case "getChannelInfo":
		// todo 获取判断资料
		result, err = nil, nil
	case "getSubscribers":
		// todo 获取订阅者
		result, err = nil, nil
	case "getBlacklist":
		// todo 获取黑名单
		result, err = nil, nil
	case "getWhitelist":
		// todo 获取白名单
		result, err = nil, nil
	case "getSystemUIDs":
		// todo 获取系统账号
		result, err = nil, nil
	}

	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(result)
}

type ChannelReq struct {
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}
