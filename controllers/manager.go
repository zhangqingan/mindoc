package controllers

import (
	"encoding/json"
	"html/template"
	"regexp"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/lifei6671/godoc/conf"
	"github.com/lifei6671/godoc/models"
	"github.com/lifei6671/godoc/utils"
)

type ManagerController struct {
	BaseController
}

func (c *ManagerController) Index() {
	c.TplName = "manager/index.tpl"
	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	c.Data["Model"] = models.NewDashboard().Query()
}

// 用户列表.
func (c *ManagerController) Users() {
	c.Prepare()
	c.TplName = "manager/users.tpl"

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	pageIndex, _ := c.GetInt("page", 0)

	members, totalCount, err := models.NewMember().FindToPager(pageIndex, 15)

	if err != nil {
		c.Data["ErrorMessage"] = err.Error()
		return
	}

	if totalCount > 0 {
		html := utils.GetPagerHtml(c.Ctx.Request.RequestURI, pageIndex, 10, int(totalCount))

		c.Data["PageHtml"] = html
	} else {
		c.Data["PageHtml"] = ""
	}

	b, err := json.Marshal(members)

	if err != nil {
		c.Data["Result"] = template.JS("[]")
	} else {
		c.Data["Result"] = template.JS(string(b))
	}
}

// 添加用户.
func (c *ManagerController) CreateMember() {
	c.Prepare()
	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	account := strings.TrimSpace(c.GetString("account"))
	password1 := strings.TrimSpace(c.GetString("password1"))
	password2 := strings.TrimSpace(c.GetString("password2"))
	email := strings.TrimSpace(c.GetString("email"))
	phone := strings.TrimSpace(c.GetString("phone"))
	role, _ := c.GetInt("role", 1)
	status, _ := c.GetInt("status", 0)

	if ok, err := regexp.MatchString(conf.RegexpAccount, account); account == "" || !ok || err != nil {
		c.JsonResult(6001, "账号只能由英文字母数字组成，且在3-50个字符")
	}
	if l := strings.Count(password1, ""); password1 == "" || l > 50 || l < 6 {
		c.JsonResult(6002, "密码必须在6-50个字符之间")
	}
	if password1 != password2 {
		c.JsonResult(6003, "确认密码不正确")
	}
	if ok, err := regexp.MatchString(conf.RegexpEmail, email); !ok || err != nil || email == "" {
		c.JsonResult(6004, "邮箱格式不正确")
	}
	if role != 0 && role != 1 && role != 2 {
		role = 1
	}
	if status != 0 && status != 1 {
		status = 0
	}

	member := models.NewMember()

	if _, err := member.FindByAccount(account); err == nil && member.MemberId > 0 {
		c.JsonResult(6005, "账号已存在")
	}

	member.Account = account
	member.Password = password1
	member.Role = role
	member.Avatar = conf.GetDefaultAvatar()
	member.CreateAt = c.Member.MemberId
	member.Email = email
	if phone != "" {
		member.Phone = phone
	}

	if err := member.Add(); err != nil {
		c.JsonResult(6006, err.Error())
	}

	c.JsonResult(0, "ok", member)
}

//更新用户状态.
func (c *ManagerController) UpdateMemberStatus() {
	c.Prepare()

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	member_id, _ := c.GetInt("member_id", 0)
	status, _ := c.GetInt("status", 0)

	if member_id <= 0 {
		c.JsonResult(6001, "参数错误")
	}
	if status != 0 && status != 1 {
		status = 0
	}
	member := models.NewMember()

	if _, err := member.Find(member_id); err != nil {
		c.JsonResult(6002, "用户不存在")
	}
	member.Status = status

	if err := member.Update(); err != nil {
		logs.Error("", err)
		c.JsonResult(6003, "用户状态设置失败")
	}
	c.JsonResult(0, "ok", member)
}

//变更用户权限.
func (c *ManagerController) ChangeMemberRole() {
	c.Prepare()

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	member_id, _ := c.GetInt("member_id", 0)
	role, _ := c.GetInt("role", 0)
	if member_id <= 0 {
		c.JsonResult(6001, "参数错误")
	}
	if role != conf.MemberAdminRole && role != conf.MemberGeneralRole {
		c.JsonResult(6001, "用户权限不正确")
	}
	member := models.NewMember()

	if _, err := member.Find(member_id); err != nil {
		c.JsonResult(6002, "用户不存在")
	}
	member.Role = role

	if err := member.Update(); err != nil {
		logs.Error("", err)
		c.JsonResult(6003, "用户权限设置失败")
	}
	member.ResolveRoleName()
	c.JsonResult(0, "ok", member)
}

func (c *ManagerController) Books() {
	c.Prepare()
	c.TplName = "manager/books.tpl"

	pageIndex, _ := c.GetInt("page", 1)

	books, totalCount, err := models.NewBookResult().FindToPager(pageIndex, conf.PageSize)

	if err != nil {
		c.Abort("500")
	}

	if totalCount > 0 {
		html := utils.GetPagerHtml(c.Ctx.Request.RequestURI, pageIndex, conf.PageSize, totalCount)

		c.Data["PageHtml"] = html
	} else {
		c.Data["PageHtml"] = ""
	}

	c.Data["Lists"] = books
}

//编辑项目
func (c *ManagerController) EditBook() {
	c.TplName = "manager/edit_book.tpl"
	identify := c.GetString(":key")

	if identify == "" {
		c.Abort("404")
	}
	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		c.Abort("500")
	}
	if c.Ctx.Input.IsPost() {

		book_name := strings.TrimSpace(c.GetString("book_name"))
		description := strings.TrimSpace(c.GetString("description", ""))
		comment_status := c.GetString("comment_status")
		tag := strings.TrimSpace(c.GetString("label"))
		order_index, _ := c.GetInt("order_index", 0)

		if strings.Count(description, "") > 500 {
			c.JsonResult(6004, "项目描述不能大于500字")
		}
		if comment_status != "open" && comment_status != "closed" && comment_status != "group_only" && comment_status != "registered_only" {
			comment_status = "closed"
		}
		if tag != "" {
			tags := strings.Split(tag, ";")
			if len(tags) > 10 {
				c.JsonResult(6005, "最多允许添加10个标签")
			}
		}

		book.BookName = book_name
		book.Description = description
		book.CommentStatus = comment_status
		book.Label = tag
		book.OrderIndex = order_index

		if err := book.Update(); err != nil {
			c.JsonResult(6006, "保存失败")
		}
		c.JsonResult(0, "ok")
	}
	if book.PrivateToken != "" {
		book.PrivateToken = c.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
	}
	c.Data["Model"] = book
}

// 删除项目.
func (c *ManagerController) DeleteBook() {
	c.Prepare()
	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	book_id, _ := c.GetInt("book_id", 0)

	if book_id <= 0 {
		c.JsonResult(6001, "参数错误")
	}
	book := models.NewBook()

	err := book.ThoroughDeleteBook(book_id)

	if err == orm.ErrNoRows {
		c.JsonResult(6002, "项目不存在")
	}
	if err != nil {
		logs.Error("DeleteBook => ", err)
		c.JsonResult(6003, "删除失败")
	}
	c.JsonResult(0, "ok")
}

// CreateToken 创建访问来令牌.
func (c *ManagerController) CreateToken() {

	action := c.GetString("action")

	identify := c.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)

	if err != nil {
		c.JsonResult(6001, "项目不存在")
	}
	if action == "create" {

		if book.PrivatelyOwned == 0 {
			c.JsonResult(6001, "公开项目不能创建阅读令牌")
		}

		book.PrivateToken = string(utils.Krand(conf.GetTokenSize(), utils.KC_RAND_KIND_ALL))
		if err := book.Update(); err != nil {
			logs.Error("生成阅读令牌失败 => ", err)
			c.JsonResult(6003, "生成阅读令牌失败")
		}
		c.JsonResult(0, "ok", c.BaseUrl()+beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken))
	} else {
		book.PrivateToken = ""
		if err := book.Update(); err != nil {
			logs.Error("CreateToken => ", err)
			c.JsonResult(6004, "删除令牌失败")
		}
		c.JsonResult(0, "ok", "")
	}
}

//项目设置.
func (c *ManagerController) Setting() {
	c.Prepare()
	c.TplName = "manager/setting.tpl"

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	options, err := models.NewOption().All()

	if c.Ctx.Input.IsPost() {
		for _, item := range options {
			item.OptionValue = c.GetString(item.OptionName)
			item.InsertOrUpdate()
		}
		c.JsonResult(0, "ok")
	}

	if err != nil {
		c.Abort("500")
	}
	c.Data["SITE_TITLE"] = c.Option["SITE_NAME"]

	for _, item := range options {
		c.Data[item.OptionName] = item
	}

}

// Transfer 转让项目.
func (c *ManagerController) Transfer() {
	c.Prepare()
	account := c.GetString("account")

	if account == "" {
		c.JsonResult(6004, "接受者账号不能为空")
	}
	member, err := models.NewMember().FindByAccount(account)

	if err != nil {
		logs.Error("FindByAccount => ", err)
		c.JsonResult(6005, "接受用户不存在")
	}
	if member.Status != 0 {
		c.JsonResult(6006, "接受用户已被禁用")
	}

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	identify := c.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		c.JsonResult(6001, err.Error())
	}
	rel, err := models.NewRelationship().FindFounder(book.BookId)

	if err != nil {
		beego.Error("FindFounder => ", err)
		c.JsonResult(6009, "查询项目创始人失败")
	}
	if member.MemberId == rel.MemberId {
		c.JsonResult(6007, "不能转让给自己")
	}

	err = models.NewRelationship().Transfer(book.BookId, rel.MemberId, member.MemberId)

	if err != nil {
		logs.Error("Transfer => ", err)
		c.JsonResult(6008, err.Error())
	}
	c.JsonResult(0, "ok")
}

func (c *ManagerController) Comments() {
	c.Prepare()
	c.TplName = "manager/comments.tpl"
	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

}

//DeleteComment 标记评论为已删除
func (c *ManagerController) DeleteComment() {
	c.Prepare()
	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}
	comment_id, _ := c.GetInt("comment_id", 0)

	if comment_id <= 0 {
		c.JsonResult(6001, "参数错误")
	}

	comment := models.NewComment()

	if _, err := comment.Find(comment_id); err != nil {
		c.JsonResult(6002, "评论不存在")
	}

	comment.Approved = 3

	if err := comment.Update("approved"); err != nil {
		c.JsonResult(6003, "删除评论失败")
	}
	c.JsonResult(0, "ok", comment)
}

//设置项目私有状态.
func (c *ManagerController) PrivatelyOwned() {

	status := c.GetString("status")
	identify := c.GetString("identify")

	if status != "open" && status != "close" {
		c.JsonResult(6003, "参数错误")
	}
	state := 0
	if status == "open" {
		state = 0
	} else {
		state = 1
	}

	if !c.Member.IsAdministrator() {
		c.Abort("403")
	}

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		c.JsonResult(6001, err.Error())
	}

	book.PrivatelyOwned = state

	logs.Info("", state, status)

	err = book.Update()

	if err != nil {
		logs.Error("PrivatelyOwned => ", err)
		c.JsonResult(6004, "保存失败")
	}
	c.JsonResult(0, "ok")
}
