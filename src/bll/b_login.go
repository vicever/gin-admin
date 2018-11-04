package bll

import (
	"context"
	"gin-admin/src/model"
	"gin-admin/src/schema"
	"gin-admin/src/util"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// 定义错误
var (
	ErrInvalidUser     = errors.New("无效的用户")
	ErrInvalidUserName = errors.New("无效的用户名")
	ErrInvalidPassword = errors.New("无效的密码")
	ErrUserDisable     = errors.New("用户被禁用")
)

// Login 登录管理
type Login struct {
	UserModel model.IUser `inject:"IUser"`
	RoleModel model.IRole `inject:"IRole"`
	MenuModel model.IMenu `inject:"IMenu"`
}

// Verify 登录验证
func (a *Login) Verify(ctx context.Context, userName, password string) (*schema.User, error) {
	user, err := a.UserModel.GetByUserName(ctx, userName, false)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, ErrInvalidUserName
	} else if user.Status != 1 {
		return nil, ErrUserDisable
	} else if user.Password != util.SHA1HashString(password) {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

// GetCurrentUserInfo 获取当前用户信息
func (a *Login) GetCurrentUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	user, err := a.UserModel.Get(ctx, userID, true)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, ErrInvalidUser
	} else if user.Status != 1 {
		return nil, ErrUserDisable
	}

	info := map[string]interface{}{
		"user_name": user.UserName,
		"real_name": user.RealName,
	}

	// 查询用户角色
	if len(user.RoleIDs) > 0 {
		roleItems, err := a.RoleModel.QuerySelect(ctx, schema.RoleSelectQueryParam{RecordIDs: user.RoleIDs})
		if err == nil && len(roleItems) > 0 {
			roleNames := make([]string, len(roleItems))
			for i, item := range roleItems {
				roleNames[i] = item.Name
			}
			info["role_names"] = roleNames
		}
	}

	return info, nil
}

// QueryCurrentUserMenus 查询当前用户菜单
func (a *Login) QueryCurrentUserMenus(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	items, err := a.MenuModel.QuerySelect(ctx, schema.MenuSelectQueryParam{
		UserID:     userID,
		Status:     1,
		SystemCode: viper.GetString("system_code"),
	})
	if err != nil {
		return nil, err
	}

	treeData := util.Slice2Tree(util.StructsToMapSlice(items), "record_id", "parent_id")
	if treeData != nil {
		a.convertMenuActionTree(&treeData)
	}

	return treeData, nil
}

// 遍历菜单树，将功能的下级菜单转换为动作数组
func (a *Login) convertMenuActionTree(child *[]map[string]interface{}) {
	for _, c := range *child {
		if util.T(c["type"]).Int() == 30 {
			children, ok := c["children"]
			if ok {
				delete(c, "children")
				childActions := *children.(*[]map[string]interface{})
				actions := make([]map[string]interface{}, len(childActions))

				for i, c := range childActions {
					actions[i] = map[string]interface{}{
						"record_id": c["record_id"],
						"code":      c["code"],
						"name":      c["name"],
						"icon":      c["icon"],
						"uri":       c["uri"],
					}
				}

				c["actions"] = actions
			}
			continue
		}

		if children, ok := c["children"]; ok {
			a.convertMenuActionTree(children.(*[]map[string]interface{}))
		}
	}
}