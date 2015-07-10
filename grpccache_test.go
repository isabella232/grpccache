package grpccache_test

import (
	"net"
	"reflect"
	"testing"
	"time"

	"sourcegraph.com/sqs/grpccache"
	"sourcegraph.com/sqs/grpccache/testpb"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestGRPCCache(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	var ts testServer
	gs := grpc.NewServer()
	testpb.RegisterTestServer(gs, &ts)
	go func() {
		if err := gs.Serve(l); err != nil {
			t.Fatal(err)
		}
	}()
	defer gs.Stop()

	cc, err := grpc.Dial(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	c := &testpb.CachedTestClient{TestClient: testpb.NewTestClient(cc), Cache: &grpccache.Cache{}}
	c.Cache.Log = true

	ctx := context.Background()

	if want := 0; len(ts.calls) != want {
		t.Errorf("got %d calls (%+v), want %d", len(ts.calls), ts.calls, want)
	}

	testNotCached := func(op *testpb.TestOp) {
		beforeNumCalls := len(ts.calls)
		r, err := c.TestMethod(ctx, op)
		if err != nil {
			t.Fatal(err)
		}
		if want := (&testpb.TestResult{X: op.A}); !reflect.DeepEqual(r, want) {
			t.Errorf("got %#v, want %#v", r, want)
		}
		if want := beforeNumCalls + 1; len(ts.calls) != want {
			t.Errorf("server did not handle call %+v (client handled it from cache), wanted it to be uncached", op)
		}
	}

	testCached := func(op *testpb.TestOp) {
		beforeNumCalls := len(ts.calls)
		r, err := c.TestMethod(ctx, op)
		if err != nil {
			t.Fatal(err)
		}
		if want := (&testpb.TestResult{X: op.A}); !reflect.DeepEqual(r, want) {
			t.Errorf("got %#v, want %#v", r, want)
		}
		if want := beforeNumCalls; len(ts.calls) != want {
			t.Errorf("server handled call %+v, wanted it to be client-cached", op)
		}
	}

	// Test caching (with no expiration)
	ts.maxAge = 999 * time.Hour
	testNotCached(&testpb.TestOp{A: 1})
	testCached(&testpb.TestOp{A: 1})
	testNotCached(&testpb.TestOp{A: 2})
	testNotCached(&testpb.TestOp{A: 2, B: []*testpb.T{{A: true}}})
	testCached(&testpb.TestOp{A: 2})
	testCached(&testpb.TestOp{A: 2, B: []*testpb.T{{A: true}}})
	testCached(&testpb.TestOp{A: 1})
	testNotCached(&testpb.TestOp{A: 3})

	// Test cache expiration
	ts.maxAge = time.Millisecond * 250
	testNotCached(&testpb.TestOp{A: 100})
	testCached(&testpb.TestOp{A: 100})
	testCached(&testpb.TestOp{A: 100})
	testNotCached(&testpb.TestOp{A: 111})
	time.Sleep(ts.maxAge)
	testNotCached(&testpb.TestOp{A: 100})
	testNotCached(&testpb.TestOp{A: 111})
	testCached(&testpb.TestOp{A: 100})
	testCached(&testpb.TestOp{A: 100})

	c.Cache.Clear()

	// Test cache max size
	c.Cache.MaxSize = 8
	testNotCached(&testpb.TestOp{A: 200})
	testCached(&testpb.TestOp{A: 200})
	testNotCached(&testpb.TestOp{A: 201})
	testCached(&testpb.TestOp{A: 201})
	testNotCached(&testpb.TestOp{A: 202}) // exceeds max size
	testNotCached(&testpb.TestOp{A: 202})
	c.Cache.MaxSize = 0
	testNotCached(&testpb.TestOp{A: 202})
	testCached(&testpb.TestOp{A: 202})
}

type testServer struct {
	calls []*testpb.TestOp

	maxAge time.Duration
}

func (s *testServer) TestMethod(ctx context.Context, op *testpb.TestOp) (*testpb.TestResult, error) {
	s.calls = append(s.calls, op)

	// Set cache control.
	if err := grpccache.SetCacheControl(ctx, grpccache.CacheControl{MaxAge: s.maxAge}); err != nil {
		return nil, err
	}

	return &testpb.TestResult{X: op.A}, nil
}
