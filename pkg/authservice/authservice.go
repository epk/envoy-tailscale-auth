package authservice

import (
	"context"
	"fmt"
	"strings"

	connect "github.com/bufbuild/connect-go"
	"google.golang.org/genproto/googleapis/rpc/status"

	envoy_auth_v3_connect "buf.build/gen/go/envoyproxy/envoy/bufbuild/connect-go/envoy/service/auth/v3/authv3connect"
	envoy_core_v3_pb "buf.build/gen/go/envoyproxy/envoy/protocolbuffers/go/envoy/config/core/v3"
	envoy_auth_v3_pb "buf.build/gen/go/envoyproxy/envoy/protocolbuffers/go/envoy/service/auth/v3"
	envoy_type_v3_pb "buf.build/gen/go/envoyproxy/envoy/protocolbuffers/go/envoy/type/v3"

	"github.com/epk/envoy-tailscale-auth/pkg/tailscale"
)

func New() envoy_auth_v3_connect.AuthorizationHandler {
	return &impl{tsClient: tailscale.NewLocalClient()}
}

type impl struct {
	tsClient tailscale.LocalClient
}

func (i *impl) Check(ctx context.Context, req *connect.Request[envoy_auth_v3_pb.CheckRequest]) (*connect.Response[envoy_auth_v3_pb.CheckResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sockAddr := req.Msg.GetAttributes().GetSource().GetAddress().GetSocketAddress()
	if sockAddr == nil {
		return connect.NewResponse(i.UnauthenticatedResponse("Forbidden, cannot determine source IP address")), nil
	}

	remoteAddr := fmt.Sprintf("%s:%d", sockAddr.GetAddress(), sockAddr.GetPortValue())

	info, err := i.tsClient.WhoIs(ctx, remoteAddr)
	if err != nil {
		return connect.NewResponse(
			i.UnauthenticatedResponse(
				fmt.Sprintf("Forbidden, can't look up tailscale metadata for %s: %v", remoteAddr, err),
			),
		), nil
	}

	// The following code is adapted from https://github.com/tailscale/tailscale/blob/c86d9f2ab1a87f3ce565457b4cb47d9c01c98c4e/cmd/nginx-auth/nginx-auth.go#L60-L92
	if len(info.Node.Tags) != 0 {
		return connect.NewResponse(
			i.UnauthenticatedResponse(
				fmt.Sprintf("Forbidden, node %s is tagged", info.Node.Hostinfo.Hostname()),
			),
		), nil
	}

	// tailnet of connected node. When accessing shared nodes, this
	// will be empty because the tailnet of the sharee is not exposed.
	var tailnet string

	if !info.Node.Hostinfo.ShareeNode() {
		var ok bool
		_, tailnet, ok = strings.Cut(info.Node.Name, info.Node.ComputedName+".")
		if !ok {
			return connect.NewResponse(
				i.UnauthenticatedResponse(
					fmt.Sprintf("Forbidden, can't extract tailnet name from hostname %q", info.Node.Name),
				),
			), nil
		}
		tailnet = strings.TrimSuffix(tailnet, ".beta.tailscale.net")
	}

	return connect.NewResponse(&envoy_auth_v3_pb.CheckResponse{
		Status: &status.Status{
			// code 0 means OK, connect does not have a constant for this
			// See: https://github.com/bufbuild/connect-go/blob/main/code.go#L35-L40
			Code: 0,
		},
		HttpResponse: &envoy_auth_v3_pb.CheckResponse_OkResponse{
			OkResponse: &envoy_auth_v3_pb.OkHttpResponse{
				Headers: []*envoy_core_v3_pb.HeaderValueOption{
					{
						AppendAction: envoy_core_v3_pb.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
						Header: &envoy_core_v3_pb.HeaderValue{
							Key:   "x-tailscale-login",
							Value: strings.Split(info.UserProfile.LoginName, "@")[0],
						},
					},
					{
						AppendAction: envoy_core_v3_pb.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
						Header: &envoy_core_v3_pb.HeaderValue{
							Key:   "x-tailscale-user",
							Value: info.UserProfile.LoginName,
						},
					},
					{
						AppendAction: envoy_core_v3_pb.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
						Header: &envoy_core_v3_pb.HeaderValue{
							Key:   "x-tailscale-name",
							Value: info.UserProfile.DisplayName,
						},
					},
					{
						AppendAction: envoy_core_v3_pb.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
						Header: &envoy_core_v3_pb.HeaderValue{
							Key:   "x-tailscale-profile-picture",
							Value: info.UserProfile.ProfilePicURL,
						},
					},
					{
						AppendAction: envoy_core_v3_pb.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
						Header: &envoy_core_v3_pb.HeaderValue{
							Key:   "x-tailscale-tailnet",
							Value: tailnet,
						},
					},
				},
			},
		},
	}), nil
}

func (i *impl) UnauthenticatedResponse(reason string) *envoy_auth_v3_pb.CheckResponse {
	return &envoy_auth_v3_pb.CheckResponse{
		Status: &status.Status{
			Code:    int32(connect.CodeUnauthenticated),
			Message: reason,
		},
		HttpResponse: &envoy_auth_v3_pb.CheckResponse_DeniedResponse{
			DeniedResponse: &envoy_auth_v3_pb.DeniedHttpResponse{
				Status: &envoy_type_v3_pb.HttpStatus{
					Code: envoy_type_v3_pb.StatusCode_Forbidden,
				},
				Body: reason,
			},
		},
	}
}
