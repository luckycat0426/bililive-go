package rpcServices

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/luckycat0426/bililive-go/src/instance"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

type Server struct {
	server *grpc.Server
}

func LoadSeverCert(path string) *credentials.TransportCredentials {
	cert, err := tls.LoadX509KeyPair(filepath.Join(path+"server-cert.pem"), filepath.Join(path+"server-key.pem"))
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(filepath.Join(path + "ca-cert.pem"))
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		err := errors.New("failed to parse root certificate")
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
	})
	return &creds
}
func StreamServerInterceptor(ctx context.Context) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		type serverStream struct {
			grpc.ServerStream
			ctx context.Context
		}
		return handler(srv, &serverStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), instance.Key, instance.GetInstance(ctx)),
		})
	}
}

func NewRpcServer(ctx context.Context) *Server {
	inst := instance.GetInstance(ctx)
	config := inst.Config
	rpcServer := grpc.NewServer(grpc.Creds(*LoadSeverCert(config.CertPath)), grpc.StreamInterceptor(StreamServerInterceptor(ctx)))
	RegisterRecordServiceServer(rpcServer, &RecordService{})
	server := &Server{
		server: rpcServer,
	}

	inst.Server = server
	return server
}
func (s *Server) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Add(1)
	lis, err := net.Listen("tcp", ":40426")
	if err != nil {
		return err
	}
	go func() {
		switch err := s.server.Serve(lis); err {
		case nil, grpc.ErrServerStopped:
		default:
			inst.Logger.Error(err)
		}
	}()
	inst.Logger.Infof("Server started:40426")
	return nil
}
func (s *Server) Close(ctx context.Context) {
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
	s.server.Stop()
	inst.Logger.Info("Server shutdown")
}
