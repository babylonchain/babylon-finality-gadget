package client

import (
	"context"
	"fmt"

	"github.com/babylonchain/babylon-finality-gadget/proto"
	"github.com/babylonchain/babylon-finality-gadget/sdk/cwclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRpcClient struct {
	client proto.FinalityGadgetClient
	conn   *grpc.ClientConn
}

func NewGRpcClient(remoteAddr string) (*GRpcClient, error) {
	conn, err := grpc.NewClient(remoteAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to build gRPC connection to %s: %w", remoteAddr, err)
	}

	gClient := &GRpcClient{
		client: proto.NewFinalityGadgetClient(conn),
		conn:   conn,
	}

	return gClient, nil
}

func (c *GRpcClient) QueryIsBlockBabylonFinalized(ctx context.Context, queryParams cwclient.L2Block) (bool, error) {
	req := &proto.QueryIsBlockBabylonFinalizedRequest{
		L2Block: &proto.L2Block{
			BlockHash:      queryParams.BlockHash,
			BlockHeight:    queryParams.BlockHeight,
			BlockTimestamp: queryParams.BlockTimestamp,
		},
	}

	resp, err := c.client.QueryIsBlockBabylonFinalized(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to query if block is finalized: %w", err)
	}

	return resp.BabylonFinalized, nil
}

func (c *GRpcClient) Close() error {
	return c.conn.Close()
}
