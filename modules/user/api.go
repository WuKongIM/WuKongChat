package user

import (
	"math/rand"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// User 用户相关API
type User struct {
	db *DB
	log.Log
	ctx *config.Context
}

// New New
func New(ctx *config.Context) *User {
	u := &User{
		ctx: ctx,
		db:  NewDB(ctx),
		Log: log.NewTLog("User"),
	}
	return u
}

// Route 路由配置
func (u *User) Route(r *wkhttp.WKHttp) {

	v := r.Group("/v1")
	{
		v.POST("/user/login", u.login) // 用户登录
		v.GET("/users/:uid", u.get)    // 根据uid查询用户信息
	}
}

// 用户资料
func (u *User) get(c *wkhttp.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.ResponseError(errors.New("uid不能为空"))
		return
	}
	model, err := u.db.queryByUID(uid)
	if err != nil {
		u.Error("查询用户资料错误", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if model == nil {
		c.ResponseError(errors.New("用户不存在"))
		return
	}
	c.Response(&userResp{
		UID:  model.UID,
		Name: model.Name,
	})
}

// 登录
func (u *User) login(c *wkhttp.Context) {
	var req loginReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if req.UID == "" || req.Token == "" {
		c.ResponseError(errors.New("uid或token不能为空！"))
		return
	}
	model, err := u.db.queryByUID(req.UID)
	if err != nil {
		u.Error("查询用户资料错误", zap.Error(err))
		c.ResponseError(err)
		return
	}
	var name string

	if model == nil {
		name = Names[rand.Intn(len(Names)-1)]
		err = u.db.insert(&userModel{
			UID:  req.UID,
			Name: name,
		})
		if err != nil {
			u.Error("新增用户错误", zap.Error(err))
			c.ResponseError(err)
			return
		}
	} else {
		name = model.Name
	}
	// 将用户信息注册到WuKongIM，如果存在则更新
	resp, err := network.Post(base.APIURL+"/user/token", []byte(util.ToJson(map[string]interface{}{
		"uid":          req.UID,
		"token":        req.Token,
		"device_level": req.DeviceLevel,
		"device_flag":  req.DeviceFlag,
	})), nil)
	if err != nil {
		u.Error("更新IM token错误", zap.Error(err))
		c.ResponseError(err)
		return
	}
	err = base.HandlerIMError(resp)
	if err != nil {
		c.ResponseError(err)
		return
	}
	var result *UpdateIMTokenResp
	if err := util.ReadJsonByByte([]byte(resp.Body), &result); err != nil {
		u.Error("解析结果错误", zap.Error(err))
		c.ResponseError(errors.New("解析结果错误"))
		return
	}
	c.Response(&userResp{
		UID:   req.UID,
		Name:  name,
		Token: req.Token,
	})
}

// UpdateIMTokenResp 更新IM Token的返回参数
type UpdateIMTokenResp struct {
	Status int `json:"status"` // 状态
}
type loginReq struct {
	UID         string `json:"uid"`
	Token       string `json:"token"`
	DeviceFlag  int    `json:"device_flag"`
	DeviceLevel int    `json:"device_level"`
}
type userResp struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Token string `json:"token,omitempty"`
}

// Names 注册用户随机名字
var Names = []string{"龚都", "黄祖", "黄祖", "黄皓", "黄琬", "黄歇", "黄权", "公孙瓒",
	"袁绍", "张角", "李儒", "高顺", "马腾", "文丑", "华雄", "颜良", "华佗",
	"左慈", "貂蝉", "司马徽", "蔡文姬", "胡车儿", "逢纪", "纪灵", "张绣", "孔融", "张鲁",
	"韩遂", "张燕", "张曼成", "审配", "黄甫嵩", "张梁", "张任", "马铁", "沪指", "辟暑大王",
	"辟尘大王", "玄鹤老", "玉兔精", "蠹妖", "蛙怪", "麋妖", "古柏老", "灵龟老", "峰五老", "赤蛇精", "虺妖", "蚖妖",
	"蝮子怪", "蝎小妖", "狐妖", "凤管娘子", "鸾萧夫人", "七情大王", "六欲大王", "三尸魔王", "阴沉魔王", "独角魔王",
	"啸风魔王", "兴云魔王", "六耳魔王", "迷识魔王", "消阳魔王", "铄阴魔王", "耗气魔王", "黑鱼精", "蜂妖", "灵鹊",
	"玄武灵", "美蔚君", "福缘君", "善庆君", "孟浪魔王", "慌张魔王", "司视魔", "司听魔", "逐香魔", "具体魔", "驰神魔",
	"逐味魔", "千里眼", "顺风耳", "金童", "玉女", "雷公", "电母", "风伯", "雨师", "游奕灵官", "翊圣真君", "大力鬼王",
	"七仙女", "太白金星", "赤脚大仙", "嫦娥", "玉兔", "吴刚", "猪八戒", "孙悟空", "唐僧", "沙悟净", "白龙马", "九天玄女",
	"九曜星", "日游神", "夜游神", "太阴星君", "太阳星君", "武德星君", "佑圣真君", "李靖", "金吒", "木吒", "哪吒",
	"巨灵神", "月老", "左辅右弼", "二郎神杨戬", "萨真人", "文昌帝君", "增长天王", "持国天王", "多闻天王", "广目天王",
	"张道陵", "许逊", "邱弘济", "葛洪", "渔人", "林黛玉", "薛宝钗", "贾宝玉", "秦可卿", "贾巧姐", "王熙凤", "史湘云",
	"妙玉", "李纨", "贾惜春", "贾探春", "贾迎春", "贾元春", "王妈妈", "西门庆", "武松", "武大郎", "宋江", "鲁智深",
	"高俅", "闻太师", "卢俊义", "吴用", "公孙胜", "关胜", "林冲", "秦明", "呼延灼", "花荣", "阮小七", "燕青",
	"皇甫端", "扈三娘", "王英", "安道全", "金大坚", "萧峰", "段誉", "童猛", "陶宗旺", "郑天寿", "王定六", "段景住",
	"寅将军", "黑熊精", "白衣秀士", "凌虚子", "黄风怪", "白骨精", "奎木狼", "金角大王", "银角大王",
}
