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
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var ServerIp string = ""

type Server struct {
	server *grpc.Server
}
type serverStream struct {
	grpc.ServerStream
	ctx context.Context
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
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ClientCAs:    certPool,
	})
	return &creds
}

func StreamServerInterceptor(ctx context.Context) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ss.SetHeader(metadata.New(map[string]string{
			"server_ip": ServerIp,
		}))
		return handler(srv, &serverStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), instance.Key, instance.GetInstance(ctx)),
		})
	}
}
func findIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock

	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}
func NewRpcServer(ctx context.Context) *Server {
	inst := instance.GetInstance(ctx)
	config := inst.Config
	res, _ := http.Get("http://ip.3322.net")
	ip, _ := ioutil.ReadAll(res.Body)
	ServerIp = findIP(string(ip))
	inst.Logger.Info("RpcServer", "Get Server IP", ServerIp)
	res.Body.Close()

	rpcServer := grpc.NewServer(grpc.Creds(*LoadSeverCert(config.CertPath)), grpc.StreamInterceptor(StreamServerInterceptor(ctx)))
	//rpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()), grpc.StreamInterceptor(StreamServerInterceptor(ctx)))
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
	var port string
	if inst.Config.RPC.Bind == "" {
		port = ":40426"
	} else {
		port = inst.Config.RPC.Bind
	}
	lis, err := net.Listen("tcp", port)
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
	inst.Logger.Infof("Server started%s", port)
	return nil
}
func (s *Server) Close(ctx context.Context) {
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
	s.server.Stop()
	inst.Logger.Info("Server shutdown")
}
