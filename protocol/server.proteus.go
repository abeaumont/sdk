package protocol

import (
	"golang.org/x/net/context"
)

type protocolServiceServer struct {
}

func NewProtocolServiceServer() *protocolServiceServer {
	return &protocolServiceServer{}
}
func (s *protocolServiceServer) Parse(ctx context.Context, in *ParseRequest) (result *ParseResponse, err error) {
	result = new(ParseResponse)
	result = Parse(in)
	return
}
func (s *protocolServiceServer) Version(ctx context.Context, in *VersionRequest) (result *VersionResponse, err error) {
	result = new(VersionResponse)
	result = Version(in)
	return
}
