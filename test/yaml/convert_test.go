package yaml_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	envoygateway_emitter "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitters/envoygateway"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/ingressnginx"
)

const testDataDir = "./testdata"

var overrideGolden = flag.Bool("override", false, "override golden files")

func TestFileConversion(t *testing.T) {
	ctx := context.Background()

	filepath.WalkDir(filepath.Join(testDataDir, "input"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err.Error())
		}
		if d.IsDir() {
			return nil
		}

		nginxProvider := ingressnginx.NewProvider(&i2gw.ProviderConf{
			ProviderSpecificFlags: map[string]map[string]string{
				ingressnginx.Name: {
					ingressnginx.NginxIngressClassFlag: "nginx",
				},
			},
		})

		err = nginxProvider.ReadResourcesFromFile(ctx, path)
		if err != nil {
			t.Fatalf("Failed to read input from file %v: %v", d.Name(), err.Error())
		}

		ir, errList := nginxProvider.ToIR()
		if len(errList) > 0 {
			t.Fatalf("unexpected errors during input conversion to ir for file %v: %v", d.Name(), errList.ToAggregate().Error())
		}

		emitter := envoygateway_emitter.NewEmitter(nil)
		gotGatewayResources, errList := emitter.Emit(ir)
		if len(errList) > 0 {
			t.Fatalf("unexpected errors during ir conversion to Gateway for file %v: %v", d.Name(), errList.ToAggregate().Error())
		}

		outputFile := filepath.Join(testDataDir, "output", d.Name())

		if *overrideGolden {
			if err := writeGatewayResourcesToFile(t, outputFile, &gotGatewayResources); err != nil {
				t.Fatalf("failed to update golden file %v: %v", outputFile, err.Error())
			}
			t.Logf("Updated golden file: %s", outputFile)
			return nil
		}

		wantGatewayResources, err := readGatewayResourcesFromFile(t, outputFile)
		if err != nil {
			t.Fatalf("failed to read wantGatewayResources from file %v: %v", outputFile, err.Error())
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.Gateways, wantGatewayResources.Gateways) {
			t.Errorf("Gateways diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.Gateways, gotGatewayResources.Gateways))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.HTTPRoutes, wantGatewayResources.HTTPRoutes) {
			t.Errorf("HTTPRoutes diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.HTTPRoutes, gotGatewayResources.HTTPRoutes))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.GRPCRoutes, wantGatewayResources.GRPCRoutes) {
			t.Errorf("GRPCRoutes diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.GRPCRoutes, gotGatewayResources.GRPCRoutes))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.TLSRoutes, wantGatewayResources.TLSRoutes) {
			t.Errorf("TLSRoutes diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.TLSRoutes, gotGatewayResources.TLSRoutes))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.TCPRoutes, wantGatewayResources.TCPRoutes) {
			t.Errorf("TCPRoutes diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.TCPRoutes, gotGatewayResources.TCPRoutes))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.UDPRoutes, wantGatewayResources.UDPRoutes) {
			t.Errorf("UDPRoutes diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.UDPRoutes, gotGatewayResources.UDPRoutes))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.BackendTLSPolicies, wantGatewayResources.BackendTLSPolicies) {
			t.Errorf("BackendTLSPolicies diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.BackendTLSPolicies, gotGatewayResources.BackendTLSPolicies))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.ReferenceGrants, wantGatewayResources.ReferenceGrants) {
			t.Errorf("ReferenceGrants diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.ReferenceGrants, gotGatewayResources.ReferenceGrants))
		}

		if !apiequality.Semantic.DeepEqual(gotGatewayResources.GatewayExtensions, wantGatewayResources.GatewayExtensions) {
			t.Errorf("GatewayExtensions diff for file %v (-want +got): %s", d.Name(), cmp.Diff(wantGatewayResources.GatewayExtensions, gotGatewayResources.GatewayExtensions))
		}
		return nil
	})
}

func readGatewayResourcesFromFile(t *testing.T, filename string) (*i2gw.GatewayResources, error) {
	t.Helper()

	stream, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %v: %w", filename, err)
	}

	unstructuredObjects, err := common.ExtractObjectsFromReader(bytes.NewReader(stream), "")
	if err != nil {
		return nil, fmt.Errorf("failed to extract objects: %w", err)
	}

	res := i2gw.GatewayResources{
		Gateways:           make(map[types.NamespacedName]gwapiv1.Gateway),
		HTTPRoutes:         make(map[types.NamespacedName]gwapiv1.HTTPRoute),
		GRPCRoutes:         make(map[types.NamespacedName]gwapiv1.GRPCRoute),
		TLSRoutes:          make(map[types.NamespacedName]gwapiv1a2.TLSRoute),
		TCPRoutes:          make(map[types.NamespacedName]gwapiv1a2.TCPRoute),
		UDPRoutes:          make(map[types.NamespacedName]gwapiv1a2.UDPRoute),
		BackendTLSPolicies: make(map[types.NamespacedName]gwapiv1.BackendTLSPolicy),
		ReferenceGrants:    make(map[types.NamespacedName]gwapiv1b1.ReferenceGrant),
		GatewayExtensions:  []unstructured.Unstructured{},
	}

	for _, obj := range unstructuredObjects {
		switch objKind := obj.GetKind(); objKind {
		case "Gateway":
			var gw gwapiv1.Gateway
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &gw); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway object: %w", err)
			}
			res.Gateways[types.NamespacedName{
				Namespace: gw.Namespace,
				Name:      gw.Name,
			}] = gw
		case "HTTPRoute":
			var httpRoute gwapiv1.HTTPRoute
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &httpRoute); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway HTTPRoute object: %w", err)
			}

			res.HTTPRoutes[types.NamespacedName{
				Namespace: httpRoute.Namespace,
				Name:      httpRoute.Name,
			}] = httpRoute
		case "GRPCRoute":
			var grpcRoute gwapiv1.GRPCRoute
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &grpcRoute); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway GRPCRoute object: %w", err)
			}

			res.GRPCRoutes[types.NamespacedName{
				Namespace: grpcRoute.Namespace,
				Name:      grpcRoute.Name,
			}] = grpcRoute
		case "TLSRoute":
			var tlsRoute gwapiv1a2.TLSRoute
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &tlsRoute); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway TLSRoute object: %w", err)
			}

			res.TLSRoutes[types.NamespacedName{
				Namespace: tlsRoute.Namespace,
				Name:      tlsRoute.Name,
			}] = tlsRoute
		case "TCPRoute":
			var tcpRoute gwapiv1a2.TCPRoute
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &tcpRoute); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway TCPRoute object: %w", err)
			}

			res.TCPRoutes[types.NamespacedName{
				Namespace: tcpRoute.Namespace,
				Name:      tcpRoute.Name,
			}] = tcpRoute
		case "UDPRoute":
			var udpRoute gwapiv1a2.UDPRoute
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &udpRoute); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway UDPRoute object: %w", err)
			}

			res.UDPRoutes[types.NamespacedName{
				Namespace: udpRoute.Namespace,
				Name:      udpRoute.Name,
			}] = udpRoute
		case "BackendTLSPolicy":
			var backendTLSPolicy gwapiv1.BackendTLSPolicy
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &backendTLSPolicy); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway BackendTLSPolicy object: %w", err)
			}

			res.BackendTLSPolicies[types.NamespacedName{
				Namespace: backendTLSPolicy.Namespace,
				Name:      backendTLSPolicy.Name,
			}] = backendTLSPolicy
		case "ReferenceGrant":
			var referenceGrant gwapiv1b1.ReferenceGrant
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &referenceGrant); err != nil {
				return nil, fmt.Errorf("failed to parse k8s gateway ReferenceGrant object: %w", err)
			}

			res.ReferenceGrants[types.NamespacedName{
				Namespace: referenceGrant.Namespace,
				Name:      referenceGrant.Name,
			}] = referenceGrant
		default:
			// Store unknown resources as GatewayExtensions
			res.GatewayExtensions = append(res.GatewayExtensions, *obj)
		}
	}
	return &res, nil
}

func writeGatewayResourcesToFile(t *testing.T, filename string, resources *i2gw.GatewayResources) error {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := gwapiv1.Install(scheme); err != nil {
		return fmt.Errorf("failed to add gwapiv1 to scheme: %w", err)
	}
	if err := gwapiv1a2.Install(scheme); err != nil {
		return fmt.Errorf("failed to add gwapiv1a2 to scheme: %w", err)
	}
	if err := gwapiv1b1.Install(scheme); err != nil {
		return fmt.Errorf("failed to add gwapiv1b1 to scheme: %w", err)
	}

	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme,
		scheme,
		json.SerializerOptions{Yaml: true, Pretty: true, Strict: true},
	)

	var buf bytes.Buffer

	for _, gw := range resources.Gateways {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&gw, &buf); err != nil {
			return fmt.Errorf("failed to encode Gateway: %w", err)
		}
	}

	for _, httpRoute := range resources.HTTPRoutes {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&httpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode HTTPRoute: %w", err)
		}
	}

	for _, grpcRoute := range resources.GRPCRoutes {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&grpcRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode GRPCRoute: %w", err)
		}
	}

	for _, tlsRoute := range resources.TLSRoutes {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&tlsRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode TLSRoute: %w", err)
		}
	}

	for _, tcpRoute := range resources.TCPRoutes {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&tcpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode TCPRoute: %w", err)
		}
	}

	for _, udpRoute := range resources.UDPRoutes {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&udpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode UDPRoute: %w", err)
		}
	}

	for _, backendTLSPolicy := range resources.BackendTLSPolicies {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&backendTLSPolicy, &buf); err != nil {
			return fmt.Errorf("failed to encode BackendTLSPolicy: %w", err)
		}
	}

	for _, refGrant := range resources.ReferenceGrants {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&refGrant, &buf); err != nil {
			return fmt.Errorf("failed to encode ReferenceGrant: %w", err)
		}
	}

	for _, ext := range resources.GatewayExtensions {
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&ext, &buf); err != nil {
			return fmt.Errorf("failed to encode GatewayExtension: %w", err)
		}
	}

	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
