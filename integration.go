package integration

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type (
	Test struct {
		t         *testing.T
		dependsOn []Dependency
	}

	TestOption interface {
		Apply(*Test)
	}

	Dependency struct {
		svc *Service
	}
)

var (
	ctx        = namespaces.NamespaceFromEnv(context.Background())
	clientOpts = containerd.WithDialOpts([]grpc.DialOption{grpc.WithTimeout(time.Second * 2), grpc.WithInsecure()})
	client     *containerd.Client
)

func createGlobalClient() error {
	if client == nil {
		c, err := containerd.New(defaults.DefaultAddress, clientOpts)
		if err != nil {
			return err
		}
		client = c
	}
	return nil
}

func NewIntegrationTest(t *testing.T, opts ...TestOption) *Test {
	err := createGlobalClient()
	if err != nil {
		t.Fatalf("[integration] coudn't create containerd client: %v", err)
	}
	it := &Test{t: t}
	for _, o := range opts {
		o.Apply(it)
	}
	return it
}

func (it *Test) Run(f func(t *testing.T)) {
	for _, dep := range it.dependsOn {
		err := dep.svc.start()
		if err != nil {
			it.t.Fatalf("[integration] couldn't create service: %v", err)
		}
		defer func() {
			err := dep.svc.stop()
			if err != nil {
				it.t.Fatalf("[integration] couldn't stop service: %v", err)
			}
		}()
		it.t.Logf("[integration] service %s is running", dep.svc.name)
	}
	f(it.t)
}

func (o Dependency) Apply(it *Test) {
	it.dependsOn = append(it.dependsOn, o)
}

func DependsOn(svc *Service) Dependency {
	return Dependency{svc: svc}
}
