package message

import (
	"embed"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
)

//go:embed sql
var sqlFS embed.FS

func init() {

	register.AddModule(func(ctx interface{}) register.Module {

		return register.Module{
			Name: "message",
			SetupAPI: func() register.APIRouter {
				return New(ctx.(*config.Context))
			},
			SQLDir: register.NewSQLFS(sqlFS),
		}
	})

	register.AddModule(func(ctx interface{}) register.Module {

		return register.Module{
			Name: "conversation",
			SetupAPI: func() register.APIRouter {
				return NewConversation(ctx.(*config.Context))
			},
		}
	})
}
