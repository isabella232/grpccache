// GENERATED CODE - DO NOT EDIT!
//
// Generated by:
//
//   go run gen_trace.go -o cache.pb.go -pkg testpb -files sourcegraph.com/sqs/grpccache/testpb@test.pb.go
//
// Called via:
//
//   go generate
//

package testpb

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sourcegraph.com/sqs/grpccache"
)

type CachedTestClient struct {
	TestClient
	Cache grpccache.Cache
}

func (s *CachedTestClient) TestMethod(ctx context.Context, in *TestOp, opts ...grpc.CallOption) (*TestResult, error) {
	var cachedResult TestResult
	cached, err := s.Cache.Get(ctx, "Test.TestMethod", in, &cachedResult)
	if err != nil {
		return nil, err
	}
	if cached {
		return &cachedResult, nil
	}
	result, err := s.TestClient.TestMethod(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := s.Cache.Store(ctx, "Test.TestMethod", in, result); err != nil {
		return nil, err
	}
	return result, nil
}
