package controllers

import (
	"encoding/base64"
	"log"
	"regexp"
	"shFreshBMS/models"
	// "strconv"

	// "github.com/astaxie/beego/utils"

	"github.com/astaxie/beego/orm"

	"github.com/astaxie/beego"
)

/*
   控制器四部曲：
   1.获取数据
   2.校验数据
   3.处理数据
   4.返回视图
   路由四部曲：
   1.
*/
// 用户模块控制器
type UserController struct {
	beego.Controller
}

//显示注册页面
func (this *UserController) ShowReg() {
	this.TplName = "register.html"
}

//处理注册数据
func (this *UserController) HandleReg() {
	userName := this.GetString("userName")
	pwd := this.GetString("password")
	//校验数据格式
	if userName == "" || pwd == "" {
		this.TplName = "register.html"
		this.Data["errmsg"] = "数据不完整，请检查！"
		return
	}

	//写入数据库，用户表
	o := orm.NewOrm()
	user := models.User{
		Name:     userName,
		PassWord: pwd,
		Power:    1, //后台用户
	}
	_, err := o.Insert(&user)
	if err != nil {
		this.TplName = "register.html"
		this.Data["errmsg"] = "注册失败，请尝试换个用户名注册！"
		return
	}
	//返回视图
	this.Ctx.WriteString("注册成功！")
}

//显示登录页面
func (this *UserController) ShowLogin() {
	userNameBase64 := this.Ctx.GetCookie("userName")
	temp, err := base64.StdEncoding.DecodeString(userNameBase64)
	if err != nil {
		log.Println("base64转换错误！")
		this.TplName = "login.html"
		return
	}

	if string(temp) == "" {
		this.Data["checked"] = ""
	} else {
		this.Data["checked"] = "checked"
	}
	this.Data["username"] = string(temp)
	this.TplName = "login.html"
}

//处理登录
func (this *UserController) HandleLogin() {
	//获得登录数据
	userName := this.GetString("userName")
	pwd := this.GetString("password")
	log.Println("用户提交用户名：", userName, "密码：:", pwd)
	//校验数据
	if userName == "" || pwd == "" {
		this.Data["errmsg"] = "登录信息不完整！"
		this.TplName = "login.html"
		return
	}
	//读取数据库并核对
	o := orm.NewOrm()
	user := models.User{Name: userName}
	err := o.Read(&user, "Name") //非主键字段需要特别指明
	log.Println("从数据库中取得用户信息：", user)
	if err != nil || user.PassWord != pwd {
		this.Data["errmsg"] = "用户名或密码错误，请重试"
		this.TplName = "login.html"
		return
	}
	if user.Power != 1 {
		this.Data["errmsg"] = "没有访问权限"
		this.TplName = "login.html"
		return
	}
	remember := this.GetString("remember")
	//保存用户名到cookie,由于cookie不能写中文，使用base64编码
	temp := base64.StdEncoding.EncodeToString([]byte(userName))
	if remember == "on" {
		this.Ctx.SetCookie("userName", temp, 24*60*60*30) //30天有效
	} else {
		this.Ctx.SetCookie("userName", temp, -1)
	}
	log.Println("cookies保存成功：", this.Ctx.GetCookie("userName"))
	//登录成功设置session
	this.SetSession("userName", userName)
	log.Println("session设置成功：", this.GetSession("userName"))
	//返回视图
	// this.Ctx.WriteString("登录成功！")
	this.Redirect("/Article/ShowArticle", 302)

}

//显示首页
func (this *UserController) ShowIndex() {
	//登录判断
	//思路：开启session存储登录信息，使用路由过滤器控制访问权限

	this.TplName = "index.html"
}

//退出登录
func (this *UserController) HandleLogout() {
	this.DelSession("userName")
	this.Redirect("/login", 302)
	log.Println("系统用户退出！")
}

//显示用户中心：用户信息
func (this *UserController) ShowUserInfo() {
	userName := GetUser(&this.Controller)
	this.Data["infoActive"] = "active"
	this.TplName = "user_center_info.html"
	//查询用户的地址信息
	o := orm.NewOrm()
	var addr models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("Isdefault", true).One(&addr)
	log.Println("查询到用户地址：", addr)
	//信息写入视图
	this.Data["phoneNum"] = addr.Phone
	this.Data["address"] = addr.Addr

}

//显示用户中心：用户订单
func (this *UserController) ShowUserOrder() {
	GetUser(&this.Controller)
	this.Data["orderActive"] = "active"
	this.TplName = "user_center_order.html"
}

//显示用户中心：用户地址
func (this *UserController) ShowUserSite() {
	userName := GetUser(&this.Controller)
	this.Data["siteActive"] = "active"
	//使用用户名查询收件地址
	o := orm.NewOrm()
	addr := models.Address{}
	err := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("Isdefault", true).One(&addr)
	if err != nil {
		log.Println("查询地址错误！", err)
		return
	}
	log.Println("查询到以下地址：", addr)
	this.Data["addr"] = addr
	this.TplName = "user_center_site.html"
}

//处理用户提交的地址信息
func (this *UserController) HandleUserSite() {
	this.TplName = "user_center_site.html"
	receiver := this.GetString("receiver")
	zipCode := this.GetString("zipCode")
	addr := this.GetString("addr")
	phone := this.GetString("phone")
	log.Println("用户提交如下信息：", receiver, zipCode, addr, phone)
	//校验数据
	//数据完整性校验
	if receiver == "" || addr == "" || zipCode == "" || phone == "" {
		log.Println("用户提交的数据不完整！")
		this.Data["errmsg"] = "信息填写不完整！"
		return
	}
	//手机号格式校验
	expr := "^1([38][0-9]|14[57]|5[^4])\\d{8}$"
	reg, err := regexp.Compile(expr)
	if err != nil {
		log.Println("正则解析错误：", err)
		return
	}
	if reg.FindString(phone) == "" {
		this.Data["errmsg"] = "手机号格式不正确，请重新输入"
		return
	}
	o := orm.NewOrm()
	var address models.Address
	address.Isdefault = true
	//先查询该用户的默认地址，没有默认地址[直接插入并设为默认]，有默认地址则先取消默认[再插入并设为默认]
	err = o.Read(&address, "Isdefault")
	if err != nil {
		log.Println("数据库未读取到默认地址：", err)
	} else {
		log.Println("数据库读取到默认地址：", address)
		address.Isdefault = false
		o.Update(&address)
	}
	//写入数据库,存在外键，一定要初始化外键值，否则插入不成功
	//再次读数据库，取得完整user作为外键
	userName := this.GetSession("userName").(string)
	user := models.User{Name: userName}
	o.Read(&user, "Name")
	//创建新地址
	address = models.Address{
		Receiver:  receiver,
		Addr:      addr,
		Zipcode:   zipCode,
		Phone:     phone,
		Isdefault: true,
		User:      &user,
	}
	//写入数据库
	_, err = o.Insert(&address)
	if err != nil {
		log.Println("写入数据库失败！", address)
		return
	}
	//提交成功后需要刷新很多信息，故直接重定向
	this.Redirect("/user/userCenterSite", 302)
}

// //封装一个函数，自动保存草稿,目前想到使用全局变量实现，但不够优雅，暂时放过
// func SaveInfo(this *beego.Controller, receiver, zipCode, addr, phone string) {
// 	this.Data["receiver"] = receiver
// 	this.Data["zipCode"] = zipCode
// 	this.Data["addr"] = addr
// 	this.Data["phone"] = phone
// 	log.Println("草稿箱：", receiver, zipCode, addr, phone)
// }

//封装函数：从session中获取用户名写入模板
//为了让函数通用性更强，直接传父类对象
func GetUser(this *beego.Controller) (uname string) {
	userName := this.GetSession("userName")
	if userName != nil {
		uname = userName.(string)
	}
	this.Data["userName"] = uname
	return
}
