package base

import (
	"fmt"
	"net/http"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/WuKongIM/WuKongIMBusinessExtra/pkg/network"
	"github.com/sendgrid/rest"
	"github.com/tidwall/gjson"
)

var APIURL = "http://175.27.245.108:15001"

// HandlerIMError 处理IM服务返回的错误信息
func HandlerIMError(resp *rest.Response) error {
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return fmt.Errorf("IM服务失败！ -> %s", resultMap["msg"])
			}
		}
		return fmt.Errorf("IM服务返回状态[%d]失败！", resp.StatusCode)
	}
	return nil
}

// SendCMD 发送CMD消息
func SendCMD(req config.MsgCMDReq) error {
	contentMap := map[string]interface{}{
		"cmd":  req.CMD,
		"type": common.CMD,
	}
	if req.Param != nil {
		contentMap["param"] = req.Param
	}
	var noPersist = 0
	if req.NoPersist {
		noPersist = 1
	}
	setting := config.Setting{
		NoUpdateConversation: true,
	}

	contentBytes := []byte(util.ToJson(contentMap))

	return SendMessage(&config.MsgSendReq{
		Header: config.MsgHeader{
			NoPersist: noPersist,
			RedDot:    0,
			SyncOnce:  1,
		},
		Setting:     setting.ToUint8(),
		FromUID:     req.FromUID,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		Subscribers: req.Subscribers,
		Payload:     contentBytes,
	})
}

// SendMessage 发送消息
func SendMessage(req *config.MsgSendReq) error {
	_, err := SendMessageWithResult(req)
	return err
}

// SendMessage 发送消息
func SendMessageWithResult(req *config.MsgSendReq) (*config.MsgSendResp, error) {
	resp, err := network.Post(APIURL+"/message/send", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return nil, err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return nil, fmt.Errorf("IM服务[SendMessage]失败！ -> %s", resultMap["msg"])
			}
		}
		return nil, fmt.Errorf("IM服务[SendMessage]返回状态[%d]失败！", resp.StatusCode)
	} else {
		dataResult := gjson.Get(resp.Body, "data")

		messageID := dataResult.Get("message_id").Int()
		messageSeq := dataResult.Get("message_seq").Int()
		clientMsgNo := dataResult.Get("client_msg_no").String()
		return &config.MsgSendResp{
			MessageID:   messageID,
			MessageSeq:  uint32(messageSeq),
			ClientMsgNo: clientMsgNo,
		}, nil
	}
}
