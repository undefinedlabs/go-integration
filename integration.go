package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"google.golang.org/grpc"
)

type (
	Test struct {
		t         *testing.T
		dependsOn []*Service
	}

	TestOption func(*Test)
)

var (
	ctx        = namespaces.NamespaceFromEnv(context.Background())
	clientOpts = containerd.WithDialOpts([]grpc.DialOption{grpc.WithTimeout(time.Second * 2), grpc.WithInsecure()})
	client     *containerd.Client
)

func createGlobalClient() error {
	if client == nil {
		address := os.Getenv("CONTAINERD_ADDRESS")
		if address == "" {
			address = defaults.DefaultAddress
		}
		c, err := containerd.New(address, clientOpts)
		if err != nil {
			return err
		}
		client = c
	}
	return nil
}

func NewIntegrationTest(t *testing.T, opts ...TestOption) *Test {
	it := &Test{t: t}
	for _, o := range opts {
		o(it)
	}
	err := createGlobalClient()
	if err != nil {
		t.Fatalf("[integration] couldn't create containerd client: %v", err)
	}
	return it
}

func (it *Test) Run(f func(ctx context.Context, t *testing.T)) {
	for _, dep := range it.dependsOn {
		running, err := dep.IsRunning()
		if err != nil {
			it.t.Fatalf("[integration] couldn't check if service is running: %v", err)
		}
		if running {
			continue
		}
		if err := dep.Start(); err != nil {
			it.t.Fatalf("[integration] couldn't create service: %v", err)
		}
		it.t.Logf("[integration] service %s is running", dep.name)
	}

	defer func() {
		for _, dep := range it.dependsOn {
			if !dep.cleanup {
				continue
			}
			err := dep.Stop()
			if err != nil {
				it.t.Fatalf("[integration] couldn't stop service: %v", err)
			}
			it.t.Logf("[integration] service %s is stopped", dep.name)
		}
	}()

	f(context.TODO(), it.t)
}

func DependsOn(svc *Service) TestOption {
	return func(test *Test) {
		test.dependsOn = append(test.dependsOn, svc)
	}
}
