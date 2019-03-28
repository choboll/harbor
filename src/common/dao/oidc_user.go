// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"fmt"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils/log"
)

// AddOIDCUser adds a oidc user
func AddOIDCUser(meta *models.OIDCUser) (int64, error) {
	now := time.Now()
	meta.CreationTime = now
	meta.UpdateTime = now
	id, err := GetOrmer().Insert(meta)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return 0, ErrDupRows
		}
		return 0, err
	}
	return id, nil
}

// GetOIDCUserByID ...
func GetOIDCUserByID(id int64) (*models.OIDCUser, error) {
	oidcUser := &models.OIDCUser{
		ID: id,
	}
	if err := GetOrmer().Read(oidcUser); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return oidcUser, nil
}

// GetUserBySub ...
func GetUserBySub(sub string) (*models.User, error) {
	var oidcUsers []models.OIDCUser
	n, err := GetOrmer().Raw(`select * from oidc_user where subiss = ? `, sub).QueryRows(&oidcUsers)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	user, err := GetUser(models.User{
		UserID: oidcUsers[0].UserID,
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("can not get user %d", oidcUsers[0].UserID)
	}

	return user, nil
}

// GetOIDCUserByUserID ...
func GetOIDCUserByUserID(userID int) (*models.OIDCUser, error) {
	var oidcUsers []models.OIDCUser
	n, err := GetOrmer().Raw(`select * from oidc_user where user_id = ? `, userID).QueryRows(&oidcUsers)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	return &oidcUsers[0], nil
}

// UpdateOIDCUser ...
func UpdateOIDCUser(oidcUser *models.OIDCUser) error {
	oidcUser.UpdateTime = time.Now()
	_, err := GetOrmer().Update(oidcUser)
	return err
}

// DeleteOIDCUser ...
func DeleteOIDCUser(id int64) error {
	_, err := GetOrmer().QueryTable(&models.OIDCUser{}).Filter("ID", id).Delete()
	return err
}

// OnBoardOIDCUser onboard OIDC user
func OnBoardOIDCUser(u models.User) error {
	o := orm.NewOrm()
	err := o.Begin()
	if err != nil {
		return err
	}
	// the password is the required attribute of user table,
	// but not required in the oidc user login scenario.
	u.Email = "odicpassword"
	var errInsert error
	userID, err := o.Insert(&u)
	if err != nil {
		errInsert = err
		log.Errorf("fail to insert user, %v", err)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			errInsert = ErrDupRows
		}
		err := o.Rollback()
		if err != nil {
			return err
		}
		return errInsert

	}
	u.OIDCUserMeta.UserID = int(userID)
	_, err = o.Insert(u.OIDCUserMeta)
	if err != nil {
		errInsert = err
		log.Errorf("fail to insert user, %v", err)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			errInsert = ErrDupRows
		}
		err := o.Rollback()
		if err != nil {
			return err
		}
		return errInsert
	}
	err = o.Commit()
	if err != nil {
		return err
	}

	return nil
}
