package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/usergroup-mservice/server/http/data"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrInvalidRuleConfig       = errors.New("invalid ruleConfig")
	ErrInvalidParam            = errors.New("invalid param")
	ErrUserGroupNotFound       = errors.New("user group not found")
	ErrUserGroupNotActive      = errors.New("user group is not active")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrUpdateNotAllowed        = errors.New("only DRAFT user groups can be updated")
	ErrEmptyExpression         = errors.New("user group expression is empty")
)

type UserGroupService interface {
	Create(ctx context.Context, req *data.CreateUserGroupRequest) (*data.UserGroupVO, error)
	Update(ctx context.Context, id int64, req *data.UpdateUserGroupRequest) (*data.UserGroupVO, error)
	Get(ctx context.Context, id int64) (*data.UserGroupVO, error)
	List(ctx context.Context, page, pageSize int, status string, userGroupID int64) (*data.UserGroupListResponse, error)
	Publish(ctx context.Context, id int64) (*data.UserGroupStatusVO, error)
	Offline(ctx context.Context, id int64) (*data.UserGroupStatusVO, error)
	EstimateSize(ctx context.Context, id int64) (*data.UserGroupCountVO, error)
	MatchUserGroup(ctx context.Context, userID, userGroupID int64) (bool, error)
}

type UserGroupServiceImpl struct {
	userGroupDao    dao.UserGroupDao
	userFullInfoDao dao.UserFullInfoDao
}

var (
	userGroupServiceOnce sync.Once
	userGroupServiceInst *UserGroupServiceImpl
)

func GetUserGroupService() *UserGroupServiceImpl {
	userGroupServiceOnce.Do(func() {
		userGroupServiceInst = &UserGroupServiceImpl{
			userGroupDao:    dao.GetUserGroupDao(),
			userFullInfoDao: dao.GetUserFullInfoDao(),
		}
	})
	return userGroupServiceInst
}

func newUserGroupService(userGroupDao dao.UserGroupDao, userFullInfoDao dao.UserFullInfoDao) *UserGroupServiceImpl {
	return &UserGroupServiceImpl{
		userGroupDao:    userGroupDao,
		userFullInfoDao: userFullInfoDao,
	}
}

func (s *UserGroupServiceImpl) Create(ctx context.Context, req *data.CreateUserGroupRequest) (*data.UserGroupVO, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRuleConfig)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidRuleConfig)
	}
	expression, err := BuildJSONPathExpression(req.RuleConfig)
	if err != nil {
		return nil, err
	}
	ruleBytes, err := json.Marshal(req.RuleConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal ruleConfig: %v", ErrInvalidRuleConfig, err)
	}
	group := &model.UserGroup{
		Name:       name,
		RuleConfig: datatypes.JSON(ruleBytes),
		Expression: expression,
		Status:     model.UserGroupStatusDraft,
	}
	id, err := s.userGroupDao.Create(ctx, group)
	if err != nil {
		log.WithContext(ctx).Errorw("create user group failed", "error", err)
		return nil, err
	}
	group.ID = id
	log.WithContext(ctx).Infow("user group created", "id", id, "name", name)
	return toUserGroupVO(group)
}

func (s *UserGroupServiceImpl) Update(ctx context.Context, id int64, req *data.UpdateUserGroupRequest) (*data.UserGroupVO, error) {
	if id <= 0 {
		return nil, ErrUserGroupNotFound
	}
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRuleConfig)
	}
	existing, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrUserGroupNotFound
	}
	if existing.Status != model.UserGroupStatusDraft {
		return nil, ErrUpdateNotAllowed
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidRuleConfig)
	}
	expression, err := BuildJSONPathExpression(req.RuleConfig)
	if err != nil {
		return nil, err
	}
	ruleBytes, err := json.Marshal(req.RuleConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal ruleConfig: %v", ErrInvalidRuleConfig, err)
	}
	existing.Name = name
	existing.RuleConfig = datatypes.JSON(ruleBytes)
	existing.Expression = expression
	if err := s.userGroupDao.Update(ctx, existing); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		log.WithContext(ctx).Errorw("update user group failed", "error", err, "id", id)
		return nil, err
	}
	updated, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrUserGroupNotFound
	}
	log.WithContext(ctx).Infow("user group updated", "id", id)
	return toUserGroupVO(updated)
}

func (s *UserGroupServiceImpl) Get(ctx context.Context, id int64) (*data.UserGroupVO, error) {
	if id <= 0 {
		return nil, ErrUserGroupNotFound
	}
	group, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrUserGroupNotFound
	}
	return toUserGroupVO(group)
}

func (s *UserGroupServiceImpl) List(ctx context.Context, page, pageSize int, status string, userGroupID int64) (*data.UserGroupListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	status = strings.TrimSpace(status)
	if status != "" && !isValidStatus(status) {
		return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidRuleConfig, status)
	}
	filter := dao.UserGroupListFilter{
		Status:      status,
		UserGroupID: userGroupID,
	}
	total, err := s.userGroupDao.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	groups, err := s.userGroupDao.List(ctx, filter, offset, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*data.UserGroupListItemVO, 0, len(groups))
	for _, g := range groups {
		items = append(items, toUserGroupListItemVO(g))
	}
	return &data.UserGroupListResponse{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    items,
	}, nil
}

func (s *UserGroupServiceImpl) Publish(ctx context.Context, id int64) (*data.UserGroupStatusVO, error) {
	group, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrUserGroupNotFound
	}
	if group.Status != model.UserGroupStatusDraft {
		return nil, fmt.Errorf("%w: publish requires DRAFT, got %s", ErrInvalidStatusTransition, group.Status)
	}
	updated, err := s.userGroupDao.UpdateStatus(ctx, id, model.UserGroupStatusActive)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		return nil, err
	}
	log.WithContext(ctx).Infow("user group published", "id", id)
	return &data.UserGroupStatusVO{
		ID:        updated.ID,
		Status:    updated.Status,
		UpdatedAt: updated.UpdatedAt.UTC(),
	}, nil
}

func (s *UserGroupServiceImpl) Offline(ctx context.Context, id int64) (*data.UserGroupStatusVO, error) {
	group, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrUserGroupNotFound
	}
	if group.Status != model.UserGroupStatusActive {
		return nil, fmt.Errorf("%w: offline requires ACTIVE, got %s", ErrInvalidStatusTransition, group.Status)
	}
	updated, err := s.userGroupDao.UpdateStatus(ctx, id, model.UserGroupStatusOffline)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		return nil, err
	}
	log.WithContext(ctx).Infow("user group offline", "id", id)
	return &data.UserGroupStatusVO{
		ID:        updated.ID,
		Status:    updated.Status,
		UpdatedAt: updated.UpdatedAt.UTC(),
	}, nil
}

func (s *UserGroupServiceImpl) EstimateSize(ctx context.Context, id int64) (*data.UserGroupCountVO, error) {
	group, err := s.userGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrUserGroupNotFound
	}
	if strings.TrimSpace(group.Expression) == "" {
		return nil, ErrEmptyExpression
	}
	count, err := s.userFullInfoDao.CountByExpression(ctx, group.Expression)
	if err != nil {
		return nil, err
	}
	return &data.UserGroupCountVO{
		UserGroupID: group.ID,
		Status:      group.Status,
		Count:       count,
		ComputedAt:  time.Now().UTC(),
	}, nil
}

func (s *UserGroupServiceImpl) MatchUserGroup(ctx context.Context, userID, userGroupID int64) (bool, error) {
	if userID <= 0 || userGroupID <= 0 {
		return false, ErrInvalidParam
	}
	group, err := s.userGroupDao.GetByID(ctx, userGroupID)
	if err != nil {
		log.WithContext(ctx).Errorw("match user group get failed", "error", err, "user_group_id", userGroupID)
		return false, err
	}
	if group == nil {
		return false, ErrUserGroupNotFound
	}
	if group.Status != model.UserGroupStatusActive {
		return false, ErrUserGroupNotActive
	}
	if strings.TrimSpace(group.Expression) == "" {
		return false, ErrEmptyExpression
	}
	matched, err := s.userFullInfoDao.ExistsByUserAndExpression(ctx, userID, group.Expression)
	if err != nil {
		log.WithContext(ctx).Errorw(
			"match user group expression failed",
			"error", err,
			"user_id", userID,
			"user_group_id", userGroupID,
		)
		return false, err
	}
	log.WithContext(ctx).Infow(
		"match user group completed",
		"user_id", userID,
		"user_group_id", userGroupID,
		"matched", matched,
	)
	return matched, nil
}

func toUserGroupVO(group *model.UserGroup) (*data.UserGroupVO, error) {
	rule, err := parseRuleConfig(group.RuleConfig)
	if err != nil {
		return nil, err
	}
	return &data.UserGroupVO{
		ID:         group.ID,
		Name:       group.Name,
		Status:     group.Status,
		RuleConfig: rule,
		CreatedAt:  group.CreatedAt.UTC(),
		UpdatedAt:  group.UpdatedAt.UTC(),
	}, nil
}

func toUserGroupListItemVO(group *model.UserGroup) *data.UserGroupListItemVO {
	return &data.UserGroupListItemVO{
		ID:     group.ID,
		Name:   group.Name,
		Status: group.Status,
	}
}

func parseRuleConfig(raw datatypes.JSON) (data.RuleConfig, error) {
	var rule data.RuleConfig
	if len(raw) == 0 {
		return rule, fmt.Errorf("%w: empty ruleConfig", ErrInvalidRuleConfig)
	}
	if err := json.Unmarshal(raw, &rule); err != nil {
		return rule, fmt.Errorf("%w: %v", ErrInvalidRuleConfig, err)
	}
	return rule, nil
}

func isValidStatus(status string) bool {
	switch status {
	case model.UserGroupStatusDraft, model.UserGroupStatusActive, model.UserGroupStatusOffline:
		return true
	default:
		return false
	}
}

var allowedFields = map[string]string{
	"registeredAt":           "number",
	"market":                 "string",
	"kycStatus":              "string",
	"totalFiatDepositUSD":    "number",
	"fiatDepositCount":       "number",
	"totalPurchaseAmountUSD": "number",
	"purchaseCount":          "number",
	"isRiskUser":             "boolean",
}

var allowedOperators = map[string]string{
	"EQ":  "==",
	"NEQ": "!=",
	"GT":  ">",
	"GTE": ">=",
	"LT":  "<",
	"LTE": "<=",
}

// BuildJSONPathExpression converts FE ruleConfig into a PostgreSQL jsonb path predicate.
func BuildJSONPathExpression(rule data.RuleConfig) (string, error) {
	logic := strings.ToUpper(strings.TrimSpace(rule.Logic))
	var joiner string
	switch logic {
	case "AND":
		joiner = " && "
	case "OR":
		joiner = " || "
	default:
		return "", fmt.Errorf("%w: logic must be AND or OR", ErrInvalidRuleConfig)
	}
	if len(rule.Conditions) == 0 {
		return "", fmt.Errorf("%w: conditions must not be empty", ErrInvalidRuleConfig)
	}
	parts := make([]string, 0, len(rule.Conditions))
	for i, cond := range rule.Conditions {
		part, err := buildConditionPredicate(cond)
		if err != nil {
			return "", fmt.Errorf("%w: condition[%d]: %v", ErrInvalidRuleConfig, i, err)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, joiner), nil
}

func buildConditionPredicate(cond data.Condition) (string, error) {
	field := strings.TrimSpace(cond.Field)
	expectedType, ok := allowedFields[field]
	if !ok {
		return "", fmt.Errorf("unsupported field %q", field)
	}
	opSymbol, ok := allowedOperators[strings.ToUpper(strings.TrimSpace(cond.Operator))]
	if !ok {
		return "", fmt.Errorf("unsupported operator %q", cond.Operator)
	}
	literal, err := formatJSONPathLiteral(cond.Value, expectedType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("$.%s %s %s", field, opSymbol, literal), nil
}

func formatJSONPathLiteral(value interface{}, expectedType string) (string, error) {
	if value == nil {
		return "", errors.New("value is required")
	}
	switch expectedType {
	case "string":
		s, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("value must be string, got %T", value)
		}
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped), nil
	case "boolean":
		switch v := value.(type) {
		case bool:
			return strconv.FormatBool(v), nil
		default:
			return "", fmt.Errorf("value must be boolean, got %T", value)
		}
	case "number":
		switch v := value.(type) {
		case float64:
			return formatNumber(v), nil
		case float32:
			return formatNumber(float64(v)), nil
		case int:
			return strconv.Itoa(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case json.Number:
			f, err := v.Float64()
			if err != nil {
				return "", fmt.Errorf("invalid number: %v", err)
			}
			return formatNumber(f), nil
		default:
			return "", fmt.Errorf("value must be number, got %T", value)
		}
	default:
		return "", fmt.Errorf("unsupported value type %q", expectedType)
	}
}

func formatNumber(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
