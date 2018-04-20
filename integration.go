package integration

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"github.com/opentracing/opentracing-go"
	"github.com/yoonitio/tracer-go/carriers"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type (
	Test struct {
		t                 *testing.T
		skipIfUnsupported bool
		dependsOn         []Dependency
	}

	TestOption interface {
		Apply(*Test)
	}

	Dependency struct {
		svc *Service
	}

	SkipOption struct{}
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
	it := &Test{t: t}
	for _, o := range opts {
		o.Apply(it)
	}
	err := createGlobalClient()
	if err != nil {
		fn := t.Fatalf
		if it.skipIfUnsupported {
			fn = t.Skipf
		}
		fn("[integration] couldn't create containerd client: %v", err)
	}
	return it
}

func (it *Test) Run(f func(ctx context.Context, t *testing.T)) {
	for _, dep := range it.dependsOn {
		err := dep.svc.Start()
		if err != nil {
			it.t.Fatalf("[integration] couldn't create service: %v", err)
		}
		defer func() {
			err := dep.svc.Stop()
			if err != nil {
				it.t.Fatalf("[integration] couldn't stop service: %v", err)
			}
		}()
		it.t.Logf("[integration] service %s is running", dep.svc.name)
	}

	tracer := opentracing.GlobalTracer()
	testContext := context.TODO()
	if tracer != nil {
		spc, err := tracer.Extract(opentracing.TextMap, carriers.NewEnvironCarrier())
		if err != nil {
			it.t.Fatalf("[integration] couldn't extract tracing context: %v", err)
		}
		sp := opentracing.GlobalTracer().StartSpan("NewIntegrationTest", opentracing.ChildOf(spc))
		defer sp.Finish()
		testContext = opentracing.ContextWithSpan(context.TODO(), sp)
	} else {
		it.t.Log("[integration] cannot find valid global tracer")
	}

	f(testContext, it.t)
}

func (o Dependency) Apply(it *Test) {
	it.dependsOn = append(it.dependsOn, o)
}

func DependsOn(svc *Service) Dependency {
	return Dependency{svc: svc}
}

func (o SkipOption) Apply(it *Test) {
	it.skipIfUnsupported = true
}

func SkipIfNoRuntimeDetected() SkipOption {
	return SkipOption{}
}
