package message

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSyncConv(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/conversation/sync", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"version":       0,
		"last_msg_seqs": "",
		"msg_count":     1,
		"login_uid":     "1",
		"device_uuid":   "1",
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
}
func TestChannelMsg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := New(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	channelID := "sll"
	channelType := 1
	loginUID := "sl"
	err = m.messageUserExtraDB.insert(&messageUserExtraModel{
		MessageID:        "1848341056756551680",
		ChannelID:        channelID,
		ChannelType:      uint8(channelType),
		UID:              loginUID,
		MessageIsDeleted: 1,
		MessageSeq:       1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/message/channel/sync", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"start_message_seq": 0,
		"end_message_seq":   100,
		"login_uid":         loginUID,
		"device_uuid":       "1",
		"pull_mode":         1,
		"limit":             1,
		"channel_type":      channelType,
		"channel_id":        channelID,
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"messages":`))
}

func TestDeleteMsg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/message", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"message_seq":  1,
		"login_uid":    "sl",
		"channel_id":   "sll",
		"channel_type": 1,
		"message_id":   "1848341056756551680",
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestRevokeMsg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/message/revoke", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"client_msg_no": "dssssdd",
		"login_uid":     "sl",
		"channel_id":    "sll",
		"channel_type":  1,
		"message_id":    "1848341056756551680",
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSyncExtra(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	m := New(ctx)
	channelId := "sll"
	channelType := common.ChannelTypePerson.Uint8()
	loginUID := "sl"

	fakeChannelID := channelId
	if channelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, channelId)
	}
	err = m.messageExtraDB.insert(&messageExtraModel{
		MessageID:   "1848341056756551680",
		MessageSeq:  1,
		ChannelID:   fakeChannelID,
		ChannelType: channelType,
		FromUID:     loginUID,
		Revoke:      1,
		Revoker:     loginUID,
		IsDeleted:   1,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/message/extra/sync", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"login_uid":     loginUID,
		"channel_id":    channelId,
		"channel_type":  channelType,
		"extra_version": 0,
		"limit":         10,
		"source":        "uuid",
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
}
