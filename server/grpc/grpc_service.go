package grpc

import (
	"context"

	"github.com/nusiss-capstone-project/usergroup-mservice/common/usergrouppb"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
)

type UsergroupService struct {
	usergrouppb.UnimplementedUsergroupServiceServer
}

func (s *UsergroupService) SayHello(ctx context.Context, in *usergrouppb.HelloRequest) (*usergrouppb.HelloResponse, error) {
	log.Logger.Infof("Received: %v", in.GetName())
	return &usergrouppb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}
