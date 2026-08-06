package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/http/data"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/service"
)

// CreateUserGroup creates a DRAFT user group.
//
// @Summary Create user group
// @Tags admin
// @Accept json
// @Produce json
// @Param body body data.CreateUserGroupRequest true "create payload"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupVO}
// @Failure 400 {object} data.BaseResponse
// @Router /usergroup-ms/v1/admin/usergroups [post]
func CreateUserGroup(c *gin.Context) {
	var req data.CreateUserGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	vo, err := service.GetUserGroupService().Create(c.Request.Context(), &req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// UpdateUserGroup updates a DRAFT user group.
//
// @Summary Update user group
// @Tags admin
// @Accept json
// @Produce json
// @Param user_group_id path int true "user group id"
// @Param body body data.UpdateUserGroupRequest true "update payload"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupVO}
// @Failure 400 {object} data.BaseResponse
// @Router /usergroup-ms/v1/admin/usergroups/{user_group_id} [put]
func UpdateUserGroup(c *gin.Context) {
	id, ok := parseUserGroupID(c)
	if !ok {
		return
	}
	var req data.UpdateUserGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	vo, err := service.GetUserGroupService().Update(c.Request.Context(), id, &req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// ListUserGroups lists user groups with filters.
//
// @Summary List user groups
// @Tags admin
// @Produce json
// @Param page query int false "page"
// @Param pageSize query int false "page size"
// @Param status query string false "status"
// @Param usergroup_id query int false "user group id"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupListResponse}
// @Router /usergroup-ms/v1/admin/usergroups [get]
func ListUserGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	var userGroupID int64
	if raw := c.Query("usergroup_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: "invalid usergroup_id"})
			return
		}
		userGroupID = parsed
	}
	vo, err := service.GetUserGroupService().List(c.Request.Context(), page, pageSize, status, userGroupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// GetUserGroup gets a user group by id.
//
// @Summary Get user group
// @Tags admin
// @Produce json
// @Param user_group_id path int true "user group id"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupVO}
// @Router /usergroup-ms/v1/admin/usergroups/{user_group_id} [get]
func GetUserGroup(c *gin.Context) {
	id, ok := parseUserGroupID(c)
	if !ok {
		return
	}
	vo, err := service.GetUserGroupService().Get(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// PublishUserGroup publishes a DRAFT user group.
//
// @Summary Publish user group
// @Tags admin
// @Produce json
// @Param user_group_id path int true "user group id"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupStatusVO}
// @Router /usergroup-ms/v1/admin/usergroups/{user_group_id}/publish [post]
func PublishUserGroup(c *gin.Context) {
	id, ok := parseUserGroupID(c)
	if !ok {
		return
	}
	vo, err := service.GetUserGroupService().Publish(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// OfflineUserGroup takes an ACTIVE user group offline.
//
// @Summary Offline user group
// @Tags admin
// @Produce json
// @Param user_group_id path int true "user group id"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupStatusVO}
// @Router /usergroup-ms/v1/admin/usergroups/{user_group_id}/offline [post]
func OfflineUserGroup(c *gin.Context) {
	id, ok := parseUserGroupID(c)
	if !ok {
		return
	}
	vo, err := service.GetUserGroupService().Offline(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

// EstimateUserGroupSize estimates matched user count.
//
// @Summary Estimate user group size
// @Tags admin
// @Produce json
// @Param user_group_id path int true "user group id"
// @Success 200 {object} data.BaseResponse{data=data.UserGroupCountVO}
// @Router /usergroup-ms/v1/admin/usergroups/{user_group_id}/count [get]
func EstimateUserGroupSize(c *gin.Context) {
	id, ok := parseUserGroupID(c)
	if !ok {
		return
	}
	vo, err := service.GetUserGroupService().EstimateSize(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: vo})
}

func parseUserGroupID(c *gin.Context) (int64, bool) {
	raw := c.Param("user_group_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: "invalid user_group_id"})
		return 0, false
	}
	return id, true
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserGroupNotFound):
		c.JSON(http.StatusNotFound, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
	case errors.Is(err, service.ErrInvalidRuleConfig),
		errors.Is(err, service.ErrInvalidStatusTransition),
		errors.Is(err, service.ErrUpdateNotAllowed),
		errors.Is(err, service.ErrEmptyExpression):
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
	}
}
