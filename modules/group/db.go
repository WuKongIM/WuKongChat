package group

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type DB struct {
	ctx     *config.Context
	session *dbr.Session
}

// NewDB NewDB
func NewDB(ctx *config.Context) *DB {
	return &DB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (db *DB) query(groupNo string) (*GroupModel, error) {
	var group *GroupModel
	_, err := db.session.Select("*").From("`group`").Where("group_no=?", groupNo).Load(&group)
	return group, err
}

func (db *DB) insert(m *GroupModel) error {
	_, err := db.session.InsertInto("group").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

type GroupModel struct {
	GroupNo string
	Name    string
	Creator string
}
