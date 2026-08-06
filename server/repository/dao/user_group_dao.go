package dao

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/model"
	"gorm.io/gorm"
)

type UserGroupListFilter struct {
	Status      string
	UserGroupID int64
}

type UserGroupDao interface {
	Create(ctx context.Context, group *model.UserGroup) (int64, error)
	Update(ctx context.Context, group *model.UserGroup) error
	GetByID(ctx context.Context, id int64) (*model.UserGroup, error)
	Count(ctx context.Context, filter UserGroupListFilter) (int64, error)
	List(ctx context.Context, filter UserGroupListFilter, offset, limit int) ([]*model.UserGroup, error)
	UpdateStatus(ctx context.Context, id int64, status string) (*model.UserGroup, error)
}

type UserGroupDaoImpl struct {
	db *gorm.DB
}

var (
	userGroupOnce sync.Once
	userGroupDao  *UserGroupDaoImpl
)

func GetUserGroupDao() *UserGroupDaoImpl {
	userGroupOnce.Do(func() {
		userGroupDao = &UserGroupDaoImpl{db: repository.DB}
	})
	return userGroupDao
}

func (d *UserGroupDaoImpl) Create(ctx context.Context, group *model.UserGroup) (int64, error) {
	now := time.Now().UTC()
	group.CreatedAt = now
	group.UpdatedAt = now
	ret := d.db.WithContext(ctx).Create(group)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to create user_group", "error", ret.Error, "name", group.Name)
		return 0, ret.Error
	}
	log.WithContext(ctx).Infow("user_group created", "id", group.ID, "name", group.Name, "status", group.Status)
	return group.ID, nil
}

func (d *UserGroupDaoImpl) Update(ctx context.Context, group *model.UserGroup) error {
	group.UpdatedAt = time.Now().UTC()
	ret := d.db.WithContext(ctx).Model(&model.UserGroup{}).
		Where("id = ?", group.ID).
		Updates(map[string]interface{}{
			"name":        group.Name,
			"rule_config": group.RuleConfig,
			"expression":  group.Expression,
			"updated_at":  group.UpdatedAt,
		})
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to update user_group", "error", ret.Error, "id", group.ID)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.WithContext(ctx).Infow("user_group updated", "id", group.ID, "name", group.Name)
	return nil
}

func (d *UserGroupDaoImpl) GetByID(ctx context.Context, id int64) (*model.UserGroup, error) {
	var group model.UserGroup
	ret := d.db.WithContext(ctx).Where("id = ?", id).First(&group)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("failed to get user_group", "error", ret.Error, "id", id)
		return nil, ret.Error
	}
	return &group, nil
}

func (d *UserGroupDaoImpl) applyListFilter(q *gorm.DB, filter UserGroupListFilter) *gorm.DB {
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.UserGroupID > 0 {
		q = q.Where("id = ?", filter.UserGroupID)
	}
	return q
}

func (d *UserGroupDaoImpl) Count(ctx context.Context, filter UserGroupListFilter) (int64, error) {
	q := d.applyListFilter(d.db.WithContext(ctx).Model(&model.UserGroup{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		log.WithContext(ctx).Errorw("failed to count user_group", "error", err)
		return 0, err
	}
	return total, nil
}

func (d *UserGroupDaoImpl) List(ctx context.Context, filter UserGroupListFilter, offset, limit int) ([]*model.UserGroup, error) {
	q := d.applyListFilter(d.db.WithContext(ctx).Model(&model.UserGroup{}), filter)
	var groups []*model.UserGroup
	ret := q.Order("id DESC").Offset(offset).Limit(limit).Find(&groups)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to list user_group", "error", ret.Error)
		return nil, ret.Error
	}
	return groups, nil
}

func (d *UserGroupDaoImpl) UpdateStatus(ctx context.Context, id int64, status string) (*model.UserGroup, error) {
	now := time.Now().UTC()
	ret := d.db.WithContext(ctx).Model(&model.UserGroup{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": now,
		})
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to update user_group status", "error", ret.Error, "id", id, "status", status)
		return nil, ret.Error
	}
	if ret.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	log.WithContext(ctx).Infow("user_group status updated", "id", id, "status", status)
	group, err := d.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return group, nil
}
