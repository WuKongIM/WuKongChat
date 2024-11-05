package webhook

import (
	"net"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhook"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Webhook Webhook
type Webhook struct {
	log.Log
	ctx *config.Context
	wkhook.UnimplementedWebhookServiceServer
	grpcServer *grpc.Server
}

// New New
func New(ctx *config.Context) *Webhook {

	return &Webhook{
		ctx: ctx,
		Log: log.NewTLog("Webhook"),
	}
}

// Route 路由配置
func (w *Webhook) Route(r *wkhttp.WKHttp) {
	r.POST("/v1/webhook", w.webhook)

	r.POST("/v2/webhook", w.webhook)

	r.POST("/v1/datasource", w.datasource)

	r.POST("/v1/webhook/message/notify", w.messageNotify) // 接受IM的消息通知,(TODO: 此接口需要与IM做安全认证)

}

func (w *Webhook) Start() error {
	w.grpcServer = grpc.NewServer()

	lis, err := net.Listen("tcp", w.ctx.GetConfig().GRPCAddr)
	if err != nil {
		return err
	}

	// 注册grpc服务
	wkhook.RegisterWebhookServiceServer(w.grpcServer, w)

	go func() {
		err = w.grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
	return nil

}

func (w *Webhook) Stop() error {
	w.grpcServer.Stop()
	return nil
}

func (w *Webhook) messageNotify(c *wkhttp.Context) {
	// todo  接收IM的消息通知
	c.ResponseOK()
}

func (w *Webhook) webhook(c *wkhttp.Context) {

	event := c.Query("event")

	data, err := c.GetRawData()
	if err != nil {
		w.Error("读取数据失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	result, err := w.handleEvent(event, data)
	if err != nil {
		w.Error("事件处理失败！", zap.Error(err), zap.String("event", event), zap.String("data", string(data)))
		c.ResponseError(err)
		return
	}
	if result != nil {
		c.Response(result)
	} else {
		c.ResponseOK()
	}

}

func (w *Webhook) handleEvent(event string, data []byte) (interface{}, error) {
	if event == "msg.offline" {
		// todo 收到离线消息
		return nil, nil
	} else if event == "user.onlinestatus" {
		// todo 在线
		return nil, nil
	} else if event == "msg.notify" {
		// todo 处理IM消息通知（所有消息）
		return nil, nil
	}
	return nil, nil
}
