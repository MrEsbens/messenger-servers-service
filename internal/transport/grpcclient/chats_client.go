package grpcclient

import (
	"context"
	"fmt"
	"time"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatsClientInterface interface {
	CreateServerChat(ctx context.Context, serverID, name, createdBy string) (string, error)
	Close() error
}

type ChatsClient struct {
	conn   *grpc.ClientConn
	client chatsv1.ChatServiceClient
}

func NewChatsClient(endpoint string) (*ChatsClient, error) {
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
		return nil, fmt.Errorf("failed to connect to chats service: %w", err)
	}

	return &ChatsClient{
		conn:   conn,
		client: chatsv1.NewChatServiceClient(conn),
	}, nil
}

func (c *ChatsClient) CreateServerChat(ctx context.Context, serverID, name, createdBy string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Создаём чат типа SERVER
	resp, err := c.client.CreateChat(ctx, &chatsv1.CreateChatRequest{
		Type:      chatsv1.ChatType_CHAT_TYPE_SERVER,
		Name:      name,
		CreatedBy: createdBy,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %w", err)
	}

	return resp.ChatId, nil
}

func (c *ChatsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
