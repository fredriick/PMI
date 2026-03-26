package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PeerServiceServer interface {
	Connect(context.Context, *ConnectRequest) (*ConnectResponse, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	ReportBandwidth(context.Context, *BandwidthReport) (*BandwidthResponse, error)
	Disconnect(context.Context, *DisconnectRequest) (*DisconnectResponse, error)
}

type UnimplementedPeerServiceServer struct{}

func (UnimplementedPeerServiceServer) Connect(context.Context, *ConnectRequest) (*ConnectResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Connect not implemented")
}

func (UnimplementedPeerServiceServer) Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}

func (UnimplementedPeerServiceServer) ReportBandwidth(context.Context, *BandwidthReport) (*BandwidthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportBandwidth not implemented")
}

func (UnimplementedPeerServiceServer) Disconnect(context.Context, *DisconnectRequest) (*DisconnectResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Disconnect not implemented")
}

func RegisterPeerServiceServer(s *grpc.Server, srv PeerServiceServer) {
	s.RegisterService(&_PeerService_serviceDesc, srv)
}

var _PeerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "proxymesh.PeerService",
	HandlerType: (*PeerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Connect",
			Handler:    _PeerService_Connect_Handler,
		},
		{
			MethodName: "Heartbeat",
			Handler:    _PeerService_Heartbeat_Handler,
		},
		{
			MethodName: "ReportBandwidth",
			Handler:    _PeerService_ReportBandwidth_Handler,
		},
		{
			MethodName: "Disconnect",
			Handler:    _PeerService_Disconnect_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "peer.proto",
}

func _PeerService_Connect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PeerServiceServer).Connect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxymesh.PeerService/Connect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PeerServiceServer).Connect(ctx, req.(*ConnectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PeerService_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PeerServiceServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxymesh.PeerService/Heartbeat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PeerServiceServer).Heartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PeerService_ReportBandwidth_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BandwidthReport)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PeerServiceServer).ReportBandwidth(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxymesh.PeerService/ReportBandwidth",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PeerServiceServer).ReportBandwidth(ctx, req.(*BandwidthReport))
	}
	return interceptor(ctx, in, info, handler)
}

func _PeerService_Disconnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisconnectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PeerServiceServer).Disconnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxymesh.PeerService/Disconnect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PeerServiceServer).Disconnect(ctx, req.(*DisconnectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

type ConnectRequest struct {
	NodeId     string  `protobuf:"varint,1,opt,name=node_id,json=nodeId"`
	Ip         string  `protobuf:"varint,2,opt,name=ip"`
	Country    string  `protobuf:"varint,3,opt,name=country"`
	City       string  `protobuf:"varint,4,opt,name=city"`
	Isp        string  `protobuf:"varint,5,opt,name=isp"`
	Os         string  `protobuf:"varint,6,opt,name=os"`
	Ipv6Subnet string  `protobuf:"varint,7,opt,name=ipv6_subnet,json=ipv6Subnet"`
	Battery    int32   `protobuf:"varint,8,opt,name=battery"`
	CpuUsage   float32 `protobuf:"fixed32,9,opt,name=cpu_usage,json=cpuUsage"`
	IsCharging bool    `protobuf:"varint,10,opt,name=is_charging,json=isCharging"`
}

type ConnectResponse struct {
	Success   bool   `protobuf:"varint,1,opt,name=success"`
	Message   string `protobuf:"varint,2,opt,name=message"`
	SessionId string `protobuf:"varint,3,opt,name=session_id,json=sessionId"`
}

type HeartbeatRequest struct {
	NodeId            string  `protobuf:"varint,1,opt,name=node_id,json=nodeId"`
	Battery           int32   `protobuf:"varint,2,opt,name=battery"`
	CpuUsage          float32 `protobuf:"fixed32,3,opt,name=cpu_usage,json=cpuUsage"`
	IsCharging        bool    `protobuf:"varint,4,opt,name=is_charging,json=isCharging"`
	BandwidthSent     int64   `protobuf:"varint,5,opt,name=bandwidth_sent,json=bandwidthSent"`
	BandwidthReceived int64   `protobuf:"varint,6,opt,name=bandwidth_received,json=bandwidthReceived"`
}

type HeartbeatResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success"`
	Message string `protobuf:"varint,2,opt,name=message"`
}

type BandwidthReport struct {
	NodeId          string `protobuf:"varint,1,opt,name=node_id,json=nodeId"`
	BytesSent       int64  `protobuf:"varint,2,opt,name=bytes_sent,json=bytesSent"`
	BytesReceived   int64  `protobuf:"varint,3,opt,name=bytes_received,json=bytesReceived"`
	DurationSeconds int64  `protobuf:"varint,4,opt,name=duration_seconds,json=durationSeconds"`
}

type BandwidthResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success"`
	Message string `protobuf:"varint,2,opt,name=message"`
}

type DisconnectRequest struct {
	NodeId string `protobuf:"varint,1,opt,name=node_id,json=nodeId"`
	Reason string `protobuf:"varint,2,opt,name=reason"`
}

type DisconnectResponse struct {
	Success bool `protobuf:"varint,1,opt,name=success"`
}
