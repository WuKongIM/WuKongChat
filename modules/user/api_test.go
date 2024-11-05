package user

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	// u := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/login", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid":          "1",
		"token":        "1",
		"device_level": 1,
		"device_flag":  0,
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"token":`))
}
func TestGetUser(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	u.db.insert(&userModel{
		UID:  "1",
		Name: "test",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/users/1", nil)
	req.Header.Set("Authorization", "Bearer 1")
	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
}
