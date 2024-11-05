package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB 用户db操作
type DB struct {
	session *dbr.Session
	ctx     *config.Context
}

// NewDB NewDB
func NewDB(ctx *config.Context) *DB {
	return &DB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}
func (d *DB) insert(m *userModel) error {
	_, err := d.session.InsertInto("user").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

// queryByUID 通过用户uid查询用户信息
func (d *DB) queryByUID(uid string) (*userModel, error) {
	var model *userModel
	_, err := d.session.Select("*").From("user").Where("uid=?", uid).Load(&model)
	return model, err
}

// ------------ model ------------

type userModel struct {
	UID  string
	Name string
}
