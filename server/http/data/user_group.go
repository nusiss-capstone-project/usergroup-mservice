package data

import "time"

type RuleConfig struct {
	Logic      string      `json:"logic" binding:"required"`
	Conditions []Condition `json:"conditions" binding:"required,min=1,dive"`
}

type Condition struct {
	Field    string      `json:"field" binding:"required"`
	Operator string      `json:"operator" binding:"required"`
	Value    interface{} `json:"value" binding:"required"`
}

type CreateUserGroupRequest struct {
	Name       string     `json:"name" binding:"required"`
	RuleConfig RuleConfig `json:"ruleConfig" binding:"required"`
}

type UpdateUserGroupRequest struct {
	Name       string     `json:"name" binding:"required"`
	RuleConfig RuleConfig `json:"ruleConfig" binding:"required"`
}

type UserGroupVO struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	RuleConfig RuleConfig `json:"ruleConfig"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type UserGroupListItemVO struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type UserGroupListResponse struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
	Total    int64                  `json:"total"`
	Items    []*UserGroupListItemVO `json:"items"`
}

type UserGroupStatusVO struct {
	ID        int64     `json:"id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserGroupCountVO struct {
	UserGroupID int64     `json:"userGroupId"`
	Status      string    `json:"status"`
	Count       int64     `json:"count"`
	ComputedAt  time.Time `json:"computedAt"`
}
