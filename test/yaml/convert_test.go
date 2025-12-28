package yaml_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

	inputDir := filepath.Join(testDataDir, "input")
	filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
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

		// Preserve directory structure in output
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			t.Fatalf("Failed to get relative path for %v: %v", path, err.Error())
		}
		outputFile := filepath.Join(testDataDir, "output", relPath)

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

		if !unstructuredSlicesEqualIgnoreOrder(gotGatewayResources.GatewayExtensions, wantGatewayResources.GatewayExtensions) {
			t.Errorf("GatewayExtensions diff for file %v (-want +got): %s", d.Name(), cmp.Diff(sortUnstructuredSlice(wantGatewayResources.GatewayExtensions), sortUnstructuredSlice(gotGatewayResources.GatewayExtensions)))
		}
		return nil
	})
}

// unstructuredSlicesEqualIgnoreOrder compares two slices of unstructured objects ignoring order.
func unstructuredSlicesEqualIgnoreOrder(a, b []unstructured.Unstructured) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort and compare
	aSorted := sortUnstructuredSlice(a)
	bSorted := sortUnstructuredSlice(b)

	return apiequality.Semantic.DeepEqual(aSorted, bSorted)
}

// sortUnstructuredSlice returns a sorted copy of the slice.
// Sorts by: GVK > Namespace > Name
func sortUnstructuredSlice(slice []unstructured.Unstructured) []unstructured.Unstructured {
	// Create a copy to avoid modifying the original
	sorted := make([]unstructured.Unstructured, len(slice))
	copy(sorted, slice)

	sort.Slice(sorted, func(i, j int) bool {
		// Sort by GVK string
		iGVK := sorted[i].GroupVersionKind().String()
		jGVK := sorted[j].GroupVersionKind().String()
		if iGVK != jGVK {
			return iGVK < jGVK
		}

		// Then by Namespace
		iNamespace := sorted[i].GetNamespace()
		jNamespace := sorted[j].GetNamespace()
		if iNamespace != jNamespace {
			return iNamespace < jNamespace
		}

		// Finally by Name
		return sorted[i].GetName() < sorted[j].GetName()
	})

	return sorted
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
			normalizeUnstructuredTypes(obj)
			res.GatewayExtensions = append(res.GatewayExtensions, *obj)
		}
	}
	return &res, nil
}

// normalizeUnstructuredTypes converts int64 to uint64 for fields that should be uint in the Go structs.
// This is necessary because YAML unmarshaling defaults to int64 for numbers, but Go structs may use uint.
func normalizeUnstructuredTypes(obj *unstructured.Unstructured) {
	content := obj.UnstructuredContent()

	// Normalize BackendTrafficPolicy rate limit requests field
	if obj.GetKind() == "BackendTrafficPolicy" {
		if spec, ok := content["spec"].(map[string]interface{}); ok {
			if rateLimit, ok := spec["rateLimit"].(map[string]interface{}); ok {
				if local, ok := rateLimit["local"].(map[string]interface{}); ok {
					if rules, ok := local["rules"].([]interface{}); ok {
						for _, rule := range rules {
							if ruleMap, ok := rule.(map[string]interface{}); ok {
								if limit, ok := ruleMap["limit"].(map[string]interface{}); ok {
									if requests, ok := limit["requests"].(int64); ok {
										limit["requests"] = uint64(requests)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func writeGatewayResourcesToFile(t *testing.T, filename string, resources *i2gw.GatewayResources) error {
	t.Helper()

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

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

	// Sort and encode Gateways
	gwKeys := make([]types.NamespacedName, 0, len(resources.Gateways))
	for k := range resources.Gateways {
		gwKeys = append(gwKeys, k)
	}
	sort.Slice(gwKeys, func(i, j int) bool {
		if gwKeys[i].Namespace != gwKeys[j].Namespace {
			return gwKeys[i].Namespace < gwKeys[j].Namespace
		}
		return gwKeys[i].Name < gwKeys[j].Name
	})
	for _, key := range gwKeys {
		gw := resources.Gateways[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&gw, &buf); err != nil {
			return fmt.Errorf("failed to encode Gateway: %w", err)
		}
	}

	// Sort and encode HTTPRoutes
	httpKeys := make([]types.NamespacedName, 0, len(resources.HTTPRoutes))
	for k := range resources.HTTPRoutes {
		httpKeys = append(httpKeys, k)
	}
	sort.Slice(httpKeys, func(i, j int) bool {
		if httpKeys[i].Namespace != httpKeys[j].Namespace {
			return httpKeys[i].Namespace < httpKeys[j].Namespace
		}
		return httpKeys[i].Name < httpKeys[j].Name
	})
	for _, key := range httpKeys {
		httpRoute := resources.HTTPRoutes[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&httpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode HTTPRoute: %w", err)
		}
	}

	// Sort and encode GRPCRoutes
	grpcKeys := make([]types.NamespacedName, 0, len(resources.GRPCRoutes))
	for k := range resources.GRPCRoutes {
		grpcKeys = append(grpcKeys, k)
	}
	sort.Slice(grpcKeys, func(i, j int) bool {
		if grpcKeys[i].Namespace != grpcKeys[j].Namespace {
			return grpcKeys[i].Namespace < grpcKeys[j].Namespace
		}
		return grpcKeys[i].Name < grpcKeys[j].Name
	})
	for _, key := range grpcKeys {
		grpcRoute := resources.GRPCRoutes[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&grpcRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode GRPCRoute: %w", err)
		}
	}

	// Sort and encode TLSRoutes
	tlsKeys := make([]types.NamespacedName, 0, len(resources.TLSRoutes))
	for k := range resources.TLSRoutes {
		tlsKeys = append(tlsKeys, k)
	}
	sort.Slice(tlsKeys, func(i, j int) bool {
		if tlsKeys[i].Namespace != tlsKeys[j].Namespace {
			return tlsKeys[i].Namespace < tlsKeys[j].Namespace
		}
		return tlsKeys[i].Name < tlsKeys[j].Name
	})
	for _, key := range tlsKeys {
		tlsRoute := resources.TLSRoutes[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&tlsRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode TLSRoute: %w", err)
		}
	}

	// Sort and encode TCPRoutes
	tcpKeys := make([]types.NamespacedName, 0, len(resources.TCPRoutes))
	for k := range resources.TCPRoutes {
		tcpKeys = append(tcpKeys, k)
	}
	sort.Slice(tcpKeys, func(i, j int) bool {
		if tcpKeys[i].Namespace != tcpKeys[j].Namespace {
			return tcpKeys[i].Namespace < tcpKeys[j].Namespace
		}
		return tcpKeys[i].Name < tcpKeys[j].Name
	})
	for _, key := range tcpKeys {
		tcpRoute := resources.TCPRoutes[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&tcpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode TCPRoute: %w", err)
		}
	}

	// Sort and encode UDPRoutes
	udpKeys := make([]types.NamespacedName, 0, len(resources.UDPRoutes))
	for k := range resources.UDPRoutes {
		udpKeys = append(udpKeys, k)
	}
	sort.Slice(udpKeys, func(i, j int) bool {
		if udpKeys[i].Namespace != udpKeys[j].Namespace {
			return udpKeys[i].Namespace < udpKeys[j].Namespace
		}
		return udpKeys[i].Name < udpKeys[j].Name
	})
	for _, key := range udpKeys {
		udpRoute := resources.UDPRoutes[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&udpRoute, &buf); err != nil {
			return fmt.Errorf("failed to encode UDPRoute: %w", err)
		}
	}

	// Sort and encode BackendTLSPolicies
	btpKeys := make([]types.NamespacedName, 0, len(resources.BackendTLSPolicies))
	for k := range resources.BackendTLSPolicies {
		btpKeys = append(btpKeys, k)
	}
	sort.Slice(btpKeys, func(i, j int) bool {
		if btpKeys[i].Namespace != btpKeys[j].Namespace {
			return btpKeys[i].Namespace < btpKeys[j].Namespace
		}
		return btpKeys[i].Name < btpKeys[j].Name
	})
	for _, key := range btpKeys {
		backendTLSPolicy := resources.BackendTLSPolicies[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&backendTLSPolicy, &buf); err != nil {
			return fmt.Errorf("failed to encode BackendTLSPolicy: %w", err)
		}
	}

	// Sort and encode ReferenceGrants
	rgKeys := make([]types.NamespacedName, 0, len(resources.ReferenceGrants))
	for k := range resources.ReferenceGrants {
		rgKeys = append(rgKeys, k)
	}
	sort.Slice(rgKeys, func(i, j int) bool {
		if rgKeys[i].Namespace != rgKeys[j].Namespace {
			return rgKeys[i].Namespace < rgKeys[j].Namespace
		}
		return rgKeys[i].Name < rgKeys[j].Name
	})
	for _, key := range rgKeys {
		refGrant := resources.ReferenceGrants[key]
		if buf.Len() > 0 {
			buf.WriteString("---\n")
		}
		if err := serializer.Encode(&refGrant, &buf); err != nil {
			return fmt.Errorf("failed to encode ReferenceGrant: %w", err)
		}
	}

	// Sort and encode GatewayExtensions
	sortedExtensions := sortUnstructuredSlice(resources.GatewayExtensions)
	for _, ext := range sortedExtensions {
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
