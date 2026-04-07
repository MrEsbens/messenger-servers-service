package grpcclient

import (
	"context"
	"fmt"
	"time"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type IdentityClientInterface interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	Close() error
}

type IdentityClient struct {
	conn   *grpc.ClientConn
	client identityv1.IdentityServiceClient
}

func NewIdentityClient(endpoint string) (*IdentityClient, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to identity service: %w", err)
	}

	return &IdentityClient{
		conn:   conn,
		client: identityv1.NewIdentityServiceClient(conn),
	}, nil
}

func (c *IdentityClient) UserExists(ctx context.Context, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.GetMe(ctx, &identityv1.GetMeRequest{UserId: userID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("identity service error: %w", err)
	}

	return resp != nil && resp.User != nil, nil
}

func (c *IdentityClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
