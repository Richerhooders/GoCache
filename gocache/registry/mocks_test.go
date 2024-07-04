package registry

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"github.com/stretchr/testify/mock"
)

type MockEtcdClient struct {
	mock.Mock
}

func (m *MockEtcdClient) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	args := m.Called(ctx, ttl)
	return args.Get(0).(*clientv3.LeaseGrantResponse), args.Error(1)
}

func (m *MockEtcdClient) KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(<-chan *clientv3.LeaseKeepAliveResponse), args.Error(1)
}

func (m *MockEtcdClient) Close() error {
	return nil
}