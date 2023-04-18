package rpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type QueenGrpcServer struct {
	listen   net.Listener
	g        *grpc.Server
	register []func(g *grpc.Server)
	ops      []grpc.ServerOption
}

func NewGrpcServer(addr string, ops ...QueenGrpcOption) (*QueenGrpcServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	ms := &QueenGrpcServer{}
	ms.listen = listen
	for _, v := range ops {
		v.Apply(ms)
	}
	server := grpc.NewServer(ms.ops...)
	ms.g = server
	return ms, nil
}

func (s *QueenGrpcServer) Run() error {
	for _, f := range s.register {
		f(s.g)
	}
	return s.g.Serve(s.listen)
}

func (s *QueenGrpcServer) Stop() {
	s.g.Stop()
}

func (s *QueenGrpcServer) Register(f func(g *grpc.Server)) {
	s.register = append(s.register, f)
}

type QueenGrpcOption interface {
	Apply(s *QueenGrpcServer)
}

type DefaultQueenGrpcOption struct {
	f func(s *QueenGrpcServer)
}

func (d *DefaultQueenGrpcOption) Apply(s *QueenGrpcServer) {
	d.f(s)
}

func WithGrpcOptions(ops ...grpc.ServerOption) QueenGrpcOption {
	return &DefaultQueenGrpcOption{
		f: func(s *QueenGrpcServer) {
			s.ops = append(s.ops, ops...)
		},
	}
}

type QueenGrpcClient struct {
	Conn *grpc.ClientConn
}

func NewGrpcClient(config *QueenGrpcClientConfig) (*QueenGrpcClient, error) {
	var ctx = context.Background()
	var dialOptions = config.dialOptions

	if config.Block {
		// 阻塞
		if config.DialTimeout > time.Duration(0) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout)
			defer cancel()
		}
		dialOptions = append(dialOptions, grpc.WithBlock())
	}
	if config.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive))
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &QueenGrpcClient{
		Conn: conn,
	}, nil
}

type QueenGrpcClientConfig struct {
	Address     string
	Block       bool
	DialTimeout time.Duration
	ReadTimeout time.Duration
	Direct      bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

func DefaultGrpcClientConfig() *QueenGrpcClientConfig {
	return &QueenGrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: time.Second * 3,
		ReadTimeout: time.Second * 2,
		Block:       true,
	}
}
