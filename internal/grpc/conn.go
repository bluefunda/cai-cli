package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/bluefunda/cai-cli/api/proto/bff"
	"github.com/bluefunda/cai-cli/internal/config"
)

// DefaultTimeout for unary RPCs.
const DefaultTimeout = 30 * time.Second

// TokenSource provides access tokens for gRPC metadata injection.
// It handles token refresh transparently.
type TokenSource struct {
	cfg         *config.Config
	refreshFunc func() (string, error)
	mu          sync.Mutex
}

// NewTokenSource creates a TokenSource that reads and refreshes tokens from config.
func NewTokenSource(cfg *config.Config, refreshFunc func() (string, error)) *TokenSource {
	return &TokenSource{cfg: cfg, refreshFunc: refreshFunc}
}

// Token returns the current access token, refreshing if expired.
func (ts *TokenSource) Token() (string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.cfg.TokenValid() {
		return ts.cfg.Auth.AccessToken, nil
	}

	if ts.refreshFunc == nil {
		if ts.cfg.Auth.AccessToken != "" {
			return ts.cfg.Auth.AccessToken, nil
		}
		return "", fmt.Errorf("not authenticated; run 'ai login'")
	}

	newToken, err := ts.refreshFunc()
	if err != nil {
		return "", fmt.Errorf("token refresh failed (run 'ai login'): %w", err)
	}
	return newToken, nil
}

// Conn wraps a gRPC client connection and provides the BFF service client.
type Conn struct {
	cc     *grpc.ClientConn
	Client pb.BFFServiceClient
}

// Dial establishes a gRPC connection to the BFF service.
// It configures auth metadata injection and default timeouts.
func Dial(target string, ts *TokenSource, opts ...grpc.DialOption) (*Conn, error) {
	if target == "" {
		return nil, fmt.Errorf("bff_url not configured; run 'ai login' or pass --bff")
	}

	defaults := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(authUnaryInterceptor(ts)),
		grpc.WithStreamInterceptor(authStreamInterceptor(ts)),
	}
	allOpts := append(defaults, opts...)

	cc, err := grpc.NewClient(target, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}
	return &Conn{
		cc:     cc,
		Client: pb.NewBFFServiceClient(cc),
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Conn) Close() error {
	return c.cc.Close()
}

// authUnaryInterceptor injects the Bearer token into unary RPC metadata.
// On Unauthenticated errors, it refreshes the token and retries once.
func authUnaryInterceptor(ts *TokenSource) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, err := attachToken(ctx, ts)
		if err != nil {
			return err
		}

		err = invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		// Retry once on Unauthenticated if we can refresh.
		if status.Code(err) == codes.Unauthenticated && ts.refreshFunc != nil {
			ts.mu.Lock()
			newToken, refreshErr := ts.refreshFunc()
			if refreshErr == nil {
				ts.cfg.Auth.AccessToken = newToken
			}
			ts.mu.Unlock()
			if refreshErr != nil {
				return err
			}
			ctx, err = attachToken(ctx, ts)
			if err != nil {
				return err
			}
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		return err
	}
}

// authStreamInterceptor injects the Bearer token into streaming RPC metadata.
func authStreamInterceptor(ts *TokenSource) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx, err := attachToken(ctx, ts)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// attachToken reads the current token and attaches it as gRPC metadata.
func attachToken(ctx context.Context, ts *TokenSource) (context.Context, error) {
	token, err := ts.Token()
	if err != nil {
		return ctx, err
	}
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md), nil
}

// ContextWithTimeout returns a context with the default unary timeout.
func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), DefaultTimeout)
}
