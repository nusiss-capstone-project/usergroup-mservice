package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/usergroup-mservice/server/config"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/http/data"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func initEnv() {
	config.Config = &config.Conf{
		LogConfig: &config.LogConfig{
			Level:    "debug",
			FilePath: "",
		},
	}
	log.InitLogger()
}

func sampleRule() data.RuleConfig {
	return data.RuleConfig{
		Logic: "AND",
		Conditions: []data.Condition{
			{Field: "kycStatus", Operator: "EQ", Value: "VERIFIED"},
			{Field: "totalFiatDepositUSD", Operator: "GTE", Value: float64(100)},
			{Field: "isRiskUser", Operator: "EQ", Value: false},
		},
	}
}

func mustRuleJSON(t *testing.T, rule data.RuleConfig) datatypes.JSON {
	t.Helper()
	b, err := json.Marshal(rule)
	require.NoError(t, err)
	return datatypes.JSON(b)
}

func TestBuildJSONPathExpression(t *testing.T) {
	tests := []struct {
		name    string
		rule    data.RuleConfig
		want    string
		wantErr bool
	}{
		{
			name: "and mix types",
			rule: sampleRule(),
			want: `$.kycStatus == "VERIFIED" && $.totalFiatDepositUSD >= 100 && $.isRiskUser == false`,
		},
		{
			name: "or logic",
			rule: data.RuleConfig{
				Logic: "OR",
				Conditions: []data.Condition{
					{Field: "market", Operator: "EQ", Value: "SG"},
					{Field: "market", Operator: "EQ", Value: "HK"},
				},
			},
			want: `$.market == "SG" || $.market == "HK"`,
		},
		{
			name: "escape quotes in string",
			rule: data.RuleConfig{
				Logic: "AND",
				Conditions: []data.Condition{
					{Field: "kycStatus", Operator: "EQ", Value: `A"B`},
				},
			},
			want: `$.kycStatus == "A\"B"`,
		},
		{
			name: "operators",
			rule: data.RuleConfig{
				Logic: "AND",
				Conditions: []data.Condition{
					{Field: "purchaseCount", Operator: "NEQ", Value: float64(0)},
					{Field: "fiatDepositCount", Operator: "GT", Value: float64(1)},
					{Field: "totalPurchaseAmountUSD", Operator: "LT", Value: float64(10.5)},
					{Field: "registeredAt", Operator: "LTE", Value: float64(1785542400)},
				},
			},
			want: `$.purchaseCount != 0 && $.fiatDepositCount > 1 && $.totalPurchaseAmountUSD < 10.5 && $.registeredAt <= 1785542400`,
		},
		{
			name:    "invalid logic",
			rule:    data.RuleConfig{Logic: "XOR", Conditions: []data.Condition{{Field: "market", Operator: "EQ", Value: "SG"}}},
			wantErr: true,
		},
		{
			name:    "empty conditions",
			rule:    data.RuleConfig{Logic: "AND"},
			wantErr: true,
		},
		{
			name:    "bad field",
			rule:    data.RuleConfig{Logic: "AND", Conditions: []data.Condition{{Field: "unknown", Operator: "EQ", Value: "x"}}},
			wantErr: true,
		},
		{
			name:    "bad operator",
			rule:    data.RuleConfig{Logic: "AND", Conditions: []data.Condition{{Field: "market", Operator: "IN", Value: "SG"}}},
			wantErr: true,
		},
		{
			name:    "type mismatch string",
			rule:    data.RuleConfig{Logic: "AND", Conditions: []data.Condition{{Field: "market", Operator: "EQ", Value: 1}}},
			wantErr: true,
		},
		{
			name:    "type mismatch bool",
			rule:    data.RuleConfig{Logic: "AND", Conditions: []data.Condition{{Field: "isRiskUser", Operator: "EQ", Value: "false"}}},
			wantErr: true,
		},
		{
			name:    "type mismatch number",
			rule:    data.RuleConfig{Logic: "AND", Conditions: []data.Condition{{Field: "purchaseCount", Operator: "EQ", Value: "1"}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildJSONPathExpression(tt.rule)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidRuleConfig))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserGroupService_Create(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)

	t.Run("success", func(t *testing.T) {
		req := &data.CreateUserGroupRequest{Name: " VIP ", RuleConfig: sampleRule()}
		ugDao.On("Create", mock.Anything, mock.MatchedBy(func(g *model.UserGroup) bool {
			return g.Name == "VIP" && g.Status == model.UserGroupStatusDraft && g.Expression != ""
		})).Return(int64(123), nil).Once()

		vo, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int64(123), vo.ID)
		assert.Equal(t, "VIP", vo.Name)
		assert.Equal(t, model.UserGroupStatusDraft, vo.Status)
		assert.Equal(t, "VERIFIED", vo.RuleConfig.Conditions[0].Value)
	})

	t.Run("nil request", func(t *testing.T) {
		_, err := svc.Create(context.Background(), nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRuleConfig))
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.Create(context.Background(), &data.CreateUserGroupRequest{Name: "  ", RuleConfig: sampleRule()})
		require.Error(t, err)
	})

	t.Run("invalid rule", func(t *testing.T) {
		_, err := svc.Create(context.Background(), &data.CreateUserGroupRequest{
			Name:       "x",
			RuleConfig: data.RuleConfig{Logic: "AND"},
		})
		require.Error(t, err)
	})

	t.Run("dao error", func(t *testing.T) {
		ugDao.On("Create", mock.Anything, mock.Anything).Return(int64(0), errors.New("db down")).Once()
		_, err := svc.Create(context.Background(), &data.CreateUserGroupRequest{Name: "x", RuleConfig: sampleRule()})
		require.Error(t, err)
	})
}

func TestUserGroupService_Update(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)
	rule := sampleRule()
	now := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		existing := &model.UserGroup{
			ID: 1, Name: "old", Status: model.UserGroupStatusDraft,
			RuleConfig: mustRuleJSON(t, rule), Expression: "x",
			CreatedAt: now, UpdatedAt: now,
		}
		ugDao.On("GetByID", mock.Anything, int64(1)).Return(existing, nil).Once()
		ugDao.On("Update", mock.Anything, mock.MatchedBy(func(g *model.UserGroup) bool {
			return g.ID == 1 && g.Name == "new"
		})).Return(nil).Once()
		updated := &model.UserGroup{
			ID: 1, Name: "new", Status: model.UserGroupStatusDraft,
			RuleConfig: mustRuleJSON(t, rule), Expression: `$.kycStatus == "VERIFIED" && $.totalFiatDepositUSD >= 100 && $.isRiskUser == false`,
			CreatedAt: now, UpdatedAt: now,
		}
		ugDao.On("GetByID", mock.Anything, int64(1)).Return(updated, nil).Once()

		vo, err := svc.Update(context.Background(), 1, &data.UpdateUserGroupRequest{Name: "new", RuleConfig: rule})
		require.NoError(t, err)
		assert.Equal(t, "new", vo.Name)
	})

	t.Run("not draft", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(2)).Return(&model.UserGroup{
			ID: 2, Status: model.UserGroupStatusActive, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Update(context.Background(), 2, &data.UpdateUserGroupRequest{Name: "n", RuleConfig: rule})
		require.ErrorIs(t, err, ErrUpdateNotAllowed)
	})

	t.Run("not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(99)).Return(nil, nil).Once()
		_, err := svc.Update(context.Background(), 99, &data.UpdateUserGroupRequest{Name: "n", RuleConfig: rule})
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		_, err := svc.Update(context.Background(), 0, &data.UpdateUserGroupRequest{Name: "n", RuleConfig: rule})
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})
}

func TestUserGroupService_GetAndList(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)
	rule := sampleRule()
	now := time.Now().UTC()
	group := &model.UserGroup{
		ID: 10, Name: "g", Status: model.UserGroupStatusDraft,
		RuleConfig: mustRuleJSON(t, rule), Expression: "$.market == \"SG\"",
		CreatedAt: now, UpdatedAt: now,
	}

	t.Run("get success", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(10)).Return(group, nil).Once()
		vo, err := svc.Get(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), vo.ID)
		assert.Equal(t, "g", vo.Name)
		assert.Equal(t, "VERIFIED", vo.RuleConfig.Conditions[0].Value)
	})

	t.Run("get not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(11)).Return(nil, nil).Once()
		_, err := svc.Get(context.Background(), 11)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})

	t.Run("list success clamps page", func(t *testing.T) {
		filter := dao.UserGroupListFilter{Status: model.UserGroupStatusDraft}
		ugDao.On("Count", mock.Anything, filter).Return(int64(1), nil).Once()
		ugDao.On("List", mock.Anything, filter, 0, 100).Return([]*model.UserGroup{group}, nil).Once()
		resp, err := svc.List(context.Background(), 0, 1000, model.UserGroupStatusDraft, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 100, resp.PageSize)
		assert.Equal(t, int64(1), resp.Total)
		require.Len(t, resp.Items, 1)
	})

	t.Run("list invalid status", func(t *testing.T) {
		_, err := svc.List(context.Background(), 1, 20, "BAD", 0)
		require.Error(t, err)
	})
}

func TestUserGroupService_PublishOfflineEstimate(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)
	rule := sampleRule()
	now := time.Now().UTC()

	t.Run("publish success", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(1)).Return(&model.UserGroup{
			ID: 1, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		ugDao.On("UpdateStatus", mock.Anything, int64(1), model.UserGroupStatusActive).Return(&model.UserGroup{
			ID: 1, Status: model.UserGroupStatusActive, UpdatedAt: now,
		}, nil).Once()
		vo, err := svc.Publish(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, model.UserGroupStatusActive, vo.Status)
	})

	t.Run("publish from active rejected", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(2)).Return(&model.UserGroup{
			ID: 2, Status: model.UserGroupStatusActive, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Publish(context.Background(), 2)
		require.ErrorIs(t, err, ErrInvalidStatusTransition)
	})

	t.Run("publish from offline rejected", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(3)).Return(&model.UserGroup{
			ID: 3, Status: model.UserGroupStatusOffline, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Publish(context.Background(), 3)
		require.ErrorIs(t, err, ErrInvalidStatusTransition)
	})

	t.Run("offline success", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(4)).Return(&model.UserGroup{
			ID: 4, Status: model.UserGroupStatusActive, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		ugDao.On("UpdateStatus", mock.Anything, int64(4), model.UserGroupStatusOffline).Return(&model.UserGroup{
			ID: 4, Status: model.UserGroupStatusOffline, UpdatedAt: now,
		}, nil).Once()
		vo, err := svc.Offline(context.Background(), 4)
		require.NoError(t, err)
		assert.Equal(t, model.UserGroupStatusOffline, vo.Status)
	})

	t.Run("offline from draft rejected", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(5)).Return(&model.UserGroup{
			ID: 5, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Offline(context.Background(), 5)
		require.ErrorIs(t, err, ErrInvalidStatusTransition)
	})

	t.Run("estimate success", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(6)).Return(&model.UserGroup{
			ID: 6, Status: model.UserGroupStatusDraft,
			Expression: `$.market == "SG"`, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		infoDao.On("CountByExpression", mock.Anything, `$.market == "SG"`).Return(int64(42), nil).Once()
		vo, err := svc.EstimateSize(context.Background(), 6)
		require.NoError(t, err)
		assert.Equal(t, int64(42), vo.Count)
		assert.Equal(t, int64(6), vo.UserGroupID)
	})

	t.Run("estimate empty expression", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(7)).Return(&model.UserGroup{
			ID: 7, Status: model.UserGroupStatusDraft, Expression: " ", RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.EstimateSize(context.Background(), 7)
		require.ErrorIs(t, err, ErrEmptyExpression)
	})

	t.Run("estimate not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(8)).Return(nil, nil).Once()
		_, err := svc.EstimateSize(context.Background(), 8)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})
}

func TestGetUserGroupServiceSingleton(t *testing.T) {
	initEnv()
	a := GetUserGroupService()
	b := GetUserGroupService()
	assert.Same(t, a, b)
}

func TestFormatJSONPathLiteral_extraTypes(t *testing.T) {
	got, err := formatJSONPathLiteral(int(7), "number")
	require.NoError(t, err)
	assert.Equal(t, "7", got)

	got, err = formatJSONPathLiteral(int64(9), "number")
	require.NoError(t, err)
	assert.Equal(t, "9", got)

	got, err = formatJSONPathLiteral(float32(1.5), "number")
	require.NoError(t, err)
	assert.Equal(t, "1.5", got)

	got, err = formatJSONPathLiteral(json.Number("12.25"), "number")
	require.NoError(t, err)
	assert.Equal(t, "12.25", got)

	_, err = formatJSONPathLiteral(nil, "string")
	require.Error(t, err)

	_, err = formatJSONPathLiteral(json.Number("bad"), "number")
	require.Error(t, err)
}

func TestParseRuleConfig_errors(t *testing.T) {
	_, err := parseRuleConfig(nil)
	require.Error(t, err)

	_, err = parseRuleConfig(datatypes.JSON([]byte(`{`)))
	require.Error(t, err)
}

func TestUserGroupService_moreBranches(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)
	rule := sampleRule()
	now := time.Now().UTC()

	t.Run("update nil req", func(t *testing.T) {
		_, err := svc.Update(context.Background(), 1, nil)
		require.ErrorIs(t, err, ErrInvalidRuleConfig)
	})

	t.Run("update empty name", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(20)).Return(&model.UserGroup{
			ID: 20, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Update(context.Background(), 20, &data.UpdateUserGroupRequest{Name: " ", RuleConfig: rule})
		require.Error(t, err)
	})

	t.Run("update invalid rule", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(21)).Return(&model.UserGroup{
			ID: 21, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		_, err := svc.Update(context.Background(), 21, &data.UpdateUserGroupRequest{
			Name: "n", RuleConfig: data.RuleConfig{Logic: "AND"},
		})
		require.Error(t, err)
	})

	t.Run("update dao error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(22)).Return(&model.UserGroup{
			ID: 22, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		ugDao.On("Update", mock.Anything, mock.Anything).Return(errors.New("db")).Once()
		_, err := svc.Update(context.Background(), 22, &data.UpdateUserGroupRequest{Name: "n", RuleConfig: rule})
		require.Error(t, err)
	})

	t.Run("get invalid id", func(t *testing.T) {
		_, err := svc.Get(context.Background(), 0)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})

	t.Run("get dao error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(23)).Return(nil, errors.New("db")).Once()
		_, err := svc.Get(context.Background(), 23)
		require.Error(t, err)
	})

	t.Run("list count dao error", func(t *testing.T) {
		ugDao.On("Count", mock.Anything, mock.Anything).Return(int64(0), errors.New("db")).Once()
		_, err := svc.List(context.Background(), 1, 20, "", 0)
		require.Error(t, err)
	})

	t.Run("list dao error", func(t *testing.T) {
		ugDao.On("Count", mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		ugDao.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db")).Once()
		_, err := svc.List(context.Background(), 1, 20, "", 0)
		require.Error(t, err)
	})

	t.Run("publish not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(24)).Return(nil, nil).Once()
		_, err := svc.Publish(context.Background(), 24)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})

	t.Run("publish dao update error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(25)).Return(&model.UserGroup{
			ID: 25, Status: model.UserGroupStatusDraft, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		ugDao.On("UpdateStatus", mock.Anything, int64(25), model.UserGroupStatusActive).Return(nil, errors.New("db")).Once()
		_, err := svc.Publish(context.Background(), 25)
		require.Error(t, err)
	})

	t.Run("offline not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(26)).Return(nil, nil).Once()
		_, err := svc.Offline(context.Background(), 26)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
	})

	t.Run("offline dao update error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(27)).Return(&model.UserGroup{
			ID: 27, Status: model.UserGroupStatusActive, RuleConfig: mustRuleJSON(t, rule), CreatedAt: now, UpdatedAt: now,
		}, nil).Once()
		ugDao.On("UpdateStatus", mock.Anything, int64(27), model.UserGroupStatusOffline).Return(nil, errors.New("db")).Once()
		_, err := svc.Offline(context.Background(), 27)
		require.Error(t, err)
	})

	t.Run("estimate dao count error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(28)).Return(&model.UserGroup{
			ID: 28, Expression: `$.market == "SG"`, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		infoDao.On("CountByExpression", mock.Anything, `$.market == "SG"`).Return(int64(0), errors.New("db")).Once()
		_, err := svc.EstimateSize(context.Background(), 28)
		require.Error(t, err)
	})
}

func TestUserGroupService_MatchUserGroup(t *testing.T) {
	initEnv()
	ugDao := new(mocks.UserGroupDao)
	infoDao := new(mocks.UserFullInfoDao)
	svc := newUserGroupService(ugDao, infoDao)
	rule := sampleRule()
	expr := `$.kycStatus == "VERIFIED" && $.totalFiatDepositUSD >= 100 && $.isRiskUser == false`

	t.Run("invalid param", func(t *testing.T) {
		matched, err := svc.MatchUserGroup(context.Background(), 0, 1)
		require.ErrorIs(t, err, ErrInvalidParam)
		assert.False(t, matched)

		matched, err = svc.MatchUserGroup(context.Background(), 1, -1)
		require.ErrorIs(t, err, ErrInvalidParam)
		assert.False(t, matched)
	})

	t.Run("group not found", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(99)).Return(nil, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 99)
		require.ErrorIs(t, err, ErrUserGroupNotFound)
		assert.False(t, matched)
	})

	t.Run("group not active", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(1)).Return(&model.UserGroup{
			ID: 1, Status: model.UserGroupStatusDraft, Expression: expr, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 1)
		require.ErrorIs(t, err, ErrUserGroupNotActive)
		assert.False(t, matched)
	})

	t.Run("offline not active", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(2)).Return(&model.UserGroup{
			ID: 2, Status: model.UserGroupStatusOffline, Expression: expr, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 2)
		require.ErrorIs(t, err, ErrUserGroupNotActive)
		assert.False(t, matched)
	})

	t.Run("empty expression", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(3)).Return(&model.UserGroup{
			ID: 3, Status: model.UserGroupStatusActive, Expression: "  ", RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 3)
		require.ErrorIs(t, err, ErrEmptyExpression)
		assert.False(t, matched)
	})

	t.Run("matched true", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(4)).Return(&model.UserGroup{
			ID: 4, Status: model.UserGroupStatusActive, Expression: expr, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		infoDao.On("ExistsByUserAndExpression", mock.Anything, int64(1001), expr).Return(true, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 4)
		require.NoError(t, err)
		assert.True(t, matched)
	})

	t.Run("matched false when profile missing or not match", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(5)).Return(&model.UserGroup{
			ID: 5, Status: model.UserGroupStatusActive, Expression: expr, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		infoDao.On("ExistsByUserAndExpression", mock.Anything, int64(1003), expr).Return(false, nil).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1003, 5)
		require.NoError(t, err)
		assert.False(t, matched)
	})

	t.Run("dao get error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(6)).Return(nil, errors.New("db")).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 6)
		require.Error(t, err)
		assert.False(t, matched)
	})

	t.Run("exists dao error", func(t *testing.T) {
		ugDao.On("GetByID", mock.Anything, int64(7)).Return(&model.UserGroup{
			ID: 7, Status: model.UserGroupStatusActive, Expression: expr, RuleConfig: mustRuleJSON(t, rule),
		}, nil).Once()
		infoDao.On("ExistsByUserAndExpression", mock.Anything, int64(1001), expr).Return(false, errors.New("db")).Once()
		matched, err := svc.MatchUserGroup(context.Background(), 1001, 7)
		require.Error(t, err)
		assert.False(t, matched)
	})
}
