package test

import (
	"fmt"
	"testing"

	"context"

	pb "github.com/tuneinc/truss/cmd/_integration-tests/middlewares/proto"
)

func TestAlwaysWrapped(t *testing.T) {
	ctx := context.Background()

	resp, err := middlewareEndpoints.AlwaysWrapped(ctx, &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.Always {
		t.Error("Always middleware did not wrap AlwaysWrap endpoint")
	}

	if !resp.NotSometimes {
		t.Error("NotSometimes middleware did not wrap AlwaysWrap endpoint")
	}
}

func TestSometimesWrapped(t *testing.T) {
	ctx := context.Background()

	resp, err := middlewareEndpoints.SometimesWrapped(ctx, &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.Always {
		t.Error("Always middleware did not wrap SometimesWrapped endpoint")
	}

	if resp.NotSometimes {
		t.Error("NotSometimes middleware did wrap SomtimesWrapped endpoint")
	}
}

func TestWrapAllLabeledExcept(t *testing.T) {
	ctx := context.Background()

	resp, err := middlewareEndpoints.LabeledTestHandler(ctx, &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	want := "LabeledTestHandler"
	if resp.Name != want {
		t.Fatal(fmt.Sprintf("want: '%s' got:'%s'", want, resp.Name))
	}
}
