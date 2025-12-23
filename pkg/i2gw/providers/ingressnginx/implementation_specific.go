package ingressnginx

import (
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func implementationSpecificHTTPPathTypeMatch(path *gwapiv1.HTTPPathMatch) {
	// Set a default path type to avoid errors during initial conversion.
	// The actual regex handling will be done by regexFeature based on use-regex annotation.
	pmPrefix := gwapiv1.PathMatchPathPrefix
	path.Type = &pmPrefix
}
