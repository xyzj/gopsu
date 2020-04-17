package microgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/xyzj/gopsu"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	kacp = keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: false,            // send pings even without active streams
	}

	kaep = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: false,           // Allow pings even when there are no active streams
	}

	kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}
)

// NewGRPCServer 初始化新的grpc服务
//
// args:
//	cafiles： 依次填入:服务端证书（必填）;服务端key（服务端单向认证/双向验证必填）;根证书（双向验证必填）;证书内服务端合法域名或ip（客户端双向验证必填）
func NewGRPCServer(cafiles ...string) (*grpc.Server, bool) {
	if len(cafiles) > 0 {
		creds, err := GetGRPCSecureConfig(cafiles...)
		if err == nil {
			return grpc.NewServer(grpc.Creds(*creds), grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp)), true
		}
	}
	return grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp)), false
}

// NewGRPCClient 初始化新的grpc客户端
func NewGRPCClient(svraddr string, cafiles ...string) (*grpc.ClientConn, error) {
	if len(cafiles) > 0 {
		creds, err := GetGRPCSecureConfig(cafiles...)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			conn, err := grpc.DialContext(
				ctx,
				svraddr,
				grpc.WithTransportCredentials(*creds),
				grpc.WithKeepaliveParams(kacp))
			cancel()
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	conn, err := grpc.DialContext(
		ctx,
		svraddr, grpc.WithInsecure(),
		grpc.WithKeepaliveParams(kacp))
	cancel()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetGRPCSecureConfig 获取grpc安全参数
//
// args:
//	cafiles： 依次填入:服务端证书（必填）;服务端key（服务端单向认证/双向验证必填）;根证书（双向验证必填）;证书内服务端合法域名或ip（客户端双向验证必填）
// return：
//	*credentials.TransportCredentials, error
func GetGRPCSecureConfig(cafiles ...string) (*credentials.TransportCredentials, error) {
	var certfile, keyfile, caroot, svrip string
	switch len(cafiles) {
	case 1: // 客户端单向认证
		certfile = cafiles[0]
		if !gopsu.IsExist(certfile) {
			return nil, fmt.Errorf("no cert file found")
		}
		creds, err := credentials.NewClientTLSFromFile(certfile, "")
		if err != nil {
			return nil, err
		}
		return &creds, nil
	case 2: // 服务端单向认证
		certfile = cafiles[0]
		keyfile = cafiles[1]
		if !gopsu.IsExist(certfile) || !gopsu.IsExist(keyfile) {
			return nil, fmt.Errorf("no cert file found")
		}
		creds, err := credentials.NewServerTLSFromFile(certfile, keyfile)
		if err != nil {
			return nil, err
		}
		return &creds, nil
	case 3, 4: // 服务端/客户端双向认证
		certfile = cafiles[0]
		keyfile = cafiles[1]
		caroot = cafiles[2]
		if len(cafiles) == 4 {
			svrip = cafiles[3]
		}
		if gopsu.IsExist(certfile) && gopsu.IsExist(keyfile) && gopsu.IsExist(caroot) {
			// Load the client certificates from disk
			certificate, err := tls.LoadX509KeyPair(certfile, keyfile)
			if err != nil {
				return nil, err
			}
			if gopsu.IsExist(caroot) {
				// Create a certificate pool from the certificate authority
				certPool := x509.NewCertPool()
				ca, err := ioutil.ReadFile(caroot)
				if err != nil {
					return nil, err
				}

				// Append the certificates from the CA
				if ok := certPool.AppendCertsFromPEM(ca); !ok {
					return nil, errors.New("failed to append ca certs")
				}

				creds := credentials.NewTLS(&tls.Config{
					ServerName:         svrip, // NOTE: this is required!
					Certificates:       []tls.Certificate{certificate},
					RootCAs:            certPool,
					InsecureSkipVerify: true,
				})
				return &creds, nil
			}
			creds, err := credentials.NewServerTLSFromFile(certfile, keyfile)
			if err != nil {
				return nil, err
			}
			return &creds, nil
		}
		return nil, fmt.Errorf("no cert,key,caroot files found")
	default:
		return nil, fmt.Errorf("no args match")
	}
}
