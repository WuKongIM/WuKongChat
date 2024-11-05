package group

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

func TestCreateGroup(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	// u := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/group/create", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"group_no":  "g1",
		"login_uid": "1",
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetGroup(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	g := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = g.db.insert(&GroupModel{
		GroupNo: "g1",
		Name:    "group1",
		Creator: "1",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/groups/g1", nil)

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"group_no":`))
}
