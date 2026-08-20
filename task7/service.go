package main

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func StartMyMicroservice(ctx context.Context, listenAddr, ACLData string) error {

	aclService := &MyMicroservice{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}

	err := parseACL(ACLData, aclService)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(makeUnaryInterceptor(aclService)),
		grpc.StreamInterceptor(makeStreamInterceptor(aclService)),
	)

	serverRegistr(grpcServer, aclService)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	go func() {
		grpcServer.Serve(listener)
	}()

	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()

	return nil
}

func parseACL(ACLData string, service *MyMicroservice) error {
	err := json.Unmarshal([]byte(ACLData), &service.ACL)
	if err != nil {
		return err
	}

	return nil
}

func serverRegistr(grpcServer *grpc.Server, aclService *MyMicroservice) {
	bizService := &BizService{}
	adminService := &AdminService{
		microservice: aclService,
	}

	RegisterBizServer(grpcServer, bizService)
	RegisterAdminServer(grpcServer, adminService)
}

func makeUnaryInterceptor(microservice *MyMicroservice) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "consumer not found")
		}

		consumers := md["consumer"]

		if len(consumers) == 0 {
			return nil, status.Error(codes.Unauthenticated, "consumer not found")
		}

		consumer := consumers[0]

		if !microservice.isAllowed(consumer, info.FullMethod) {
			return nil, status.Error(codes.Unauthenticated, "access denied")
		}

		microservice.countRequest(consumer, info.FullMethod)

		event, err := makeEvent(consumer, info.FullMethod, ctx)
		if err != nil {
			return nil, err
		}

		microservice.sendToLogSubscribers(&event)

		return handler(ctx, req)
	}
}

func makeStreamInterceptor(microservice *MyMicroservice) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "consumer not found")
		}

		consumers := md["consumer"]

		if len(consumers) == 0 {
			return status.Error(codes.Unauthenticated, "consumer not found")
		}

		consumer := consumers[0]

		if !microservice.isAllowed(consumer, info.FullMethod) {
			return status.Error(codes.Unauthenticated, "access denied")
		}

		microservice.countRequest(consumer, info.FullMethod)

		event, err := makeEvent(consumer, info.FullMethod, ctx)
		if err != nil {
			return err
		}

		microservice.sendToLogSubscribers(&event)

		return handler(srv, ss)
	}
}

func makeEvent(consumer string, method string, ctx context.Context) (Event, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return Event{}, status.Error(codes.Unknown, "peer not found")
	}

	host := p.Addr.String()

	return Event{
		Timestamp: time.Now().Unix(),
		Consumer:  consumer,
		Method:    method,
		Host:      host,
	}, nil
}
