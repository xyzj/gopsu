package microgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/xyzj/gopsu"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewGRPCServer 初始化新的grpc服务
func NewGRPCServer(cafiles ...string) (*grpc.Server, error) {
	if len(cafiles) > 0 {
		creds, err := GetGRPCSecureConfig(cafiles...)
		if err != nil {
			return nil, err
		}
		return grpc.NewServer(grpc.Creds(*creds)), nil
	}
	return grpc.NewServer(), nil
}

// NewGRPCClient 初始化新的grpc客户端
func NewGRPCClient(svraddr string, cafiles ...string) (*grpc.ClientConn, error) {
	if len(cafiles) > 0 {
		creds, err := GetGRPCSecureConfig(cafiles...)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		conn, err := grpc.DialContext(ctx, svraddr, grpc.WithTransportCredentials(*creds))
		cancel()
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	conn, err := grpc.DialContext(ctx, svraddr, grpc.WithInsecure())
	cancel()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetGRPCSecureConfig 获取grpc安全参数
//
// args:
//	cafiles： 依次填入，服务端证书（必填），服务端key（服务端单向认证/双向验证必填），根证书（双向验证必填），证书内服务端合法域名或ip（双向验证必填）
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
	case 4: // 客户端/服务端双向认证
		certfile = cafiles[0]
		keyfile = cafiles[1]
		caroot = cafiles[2]
		svrip = cafiles[3]
		if gopsu.IsExist(certfile) && gopsu.IsExist(keyfile) && gopsu.IsExist(caroot) && gopsu.IsExist(svrip) {
			// Load the client certificates from disk
			certificate, err := tls.LoadX509KeyPair(certfile, keyfile)
			if err != nil {
				return nil, err
			}

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
				ServerName:   svrip, // NOTE: this is required!
				Certificates: []tls.Certificate{certificate},
				RootCAs:      certPool,
			})
			return &creds, nil
		}
		return nil, fmt.Errorf("no cert,key,caroot files found")
	default:
		return nil, fmt.Errorf("no args match")
	}
}
