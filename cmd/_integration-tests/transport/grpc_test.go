package test

import (
	"strings"
	"testing"
	"time"

	"context"
	"google.golang.org/grpc"

	pb "github.com/tuneinc/truss/cmd/_integration-tests/transport/proto"
	grpcclient "github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service/svc/client/grpc"
)

var grpcAddr string

func init() { _ = grpcAddr }

func TestCtxToCtxViaGRPCMetadata(t *testing.T) {
	var req pb.MetaRequest
	var key, value = "Truss-Auth-Header", "SECRET"
	req.Key = key

	// Create a new client telling it to send "Truss-Auth-Header" as a header
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	svcgrpc, err := grpcclient.New(conn,
		grpcclient.CtxValuesToSend(key))
	if err != nil {
		t.Fatalf("failed to create grpcclient: %q", err)
	}

	// Create a context with the header key and valu
	ctx := context.WithValue(context.Background(), key, value)

	// send the context
	resp, err := svcgrpc.CtxToCtx(ctx, &req)
	if err != nil {
		t.Fatalf("grpcclient returned error: %q", err)
	}

	if resp.V != value {
		t.Fatalf("Expect: %q, got %q", value, resp.V)
	}

	// Test the key in the ToLower format as metadata sends that over the wire
	req.Key = strings.ToLower(req.Key)
	// send the context
	resp, err = svcgrpc.CtxToCtx(ctx, &req)
	if err != nil {
		t.Fatalf("grpcclient returned error: %q", err)
	}

	if resp.V != value {
		t.Fatalf("Expect: %q, got %q", value, resp.V)
	}
}

func TestHTTPErrorStatusCodeAndHeadersWithGRPC(t *testing.T) {
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	svcgrpc, err := grpcclient.New(conn)
	if err != nil {
		t.Fatalf("failed to create grpcclient: %q", err)
	}

	_, err = svcgrpc.StatusCodeAndHeaders(context.Background(), &pb.Empty{})
	if err == nil {
		t.Fatalf("Expected error")
	}
}
