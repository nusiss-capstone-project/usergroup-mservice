package grpc

import (
	"context"
	"errors"

	"github.com/nusiss-capstone-project/usergroup-mservice/common/usergrouppb"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/service"
)

type UsergroupService struct {
	usergrouppb.UnimplementedUsergroupServiceServer
}

func (s *UsergroupService) MatchUserGroup(
	ctx context.Context,
	in *usergrouppb.MatchUserGroupRequest,
) (*usergrouppb.MatchUserGroupResponse, error) {
	matched, err := service.GetUserGroupService().MatchUserGroup(ctx, in.GetUserId(), in.GetUserGroupId())
	if err != nil {
		code, message := mapMatchError(err)
		log.WithContext(ctx).Warnw(
			"MatchUserGroup failed",
			"user_id", in.GetUserId(),
			"user_group_id", in.GetUserGroupId(),
			"code", code.String(),
			"error", err,
		)
		return &usergrouppb.MatchUserGroupResponse{
			Matched: false,
			BaseResponseInfo: &usergrouppb.BaseResponseInfo{
				Code:    code,
				Message: message,
			},
		}, nil
	}
	return &usergrouppb.MatchUserGroupResponse{
		Matched: matched,
		BaseResponseInfo: &usergrouppb.BaseResponseInfo{
			Code:    usergrouppb.ErrorCode_OK,
			Message: usergrouppb.ErrorCode_OK.String(),
		},
	}, nil
}

func mapMatchError(err error) (usergrouppb.ErrorCode, string) {
	switch {
	case errors.Is(err, service.ErrInvalidParam), errors.Is(err, service.ErrEmptyExpression):
		return usergrouppb.ErrorCode_INVALID_PARAM, err.Error()
	case errors.Is(err, service.ErrUserGroupNotFound):
		return usergrouppb.ErrorCode_DATA_NOT_EXIST, err.Error()
	case errors.Is(err, service.ErrUserGroupNotActive):
		return usergrouppb.ErrorCode_USER_GROUP_NOT_ACTIVE, err.Error()
	default:
		return usergrouppb.ErrorCode_UNKNOWN_ERROR, err.Error()
	}
}
