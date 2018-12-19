package kubectl

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

// SetKubernetesDefaults sets default values on the provided client config for accessing the
// Kubernetes API or returns an error if any of the defaults are impossible or invalid.
// NOTE: Originally copied from here:
//   https://github.com/kubernetes/kubernetes/blob/ddf47ac13c1a9483ea035a79cd7c10005ff21a6d/pkg/kubectl/cmd/util/kubectl_match_version.go#L113-L130
func SetKubernetesDefaults(config *rest.Config) error {
	// TODO remove this hack.  This is allowing the GetOptions to be serialized.
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}

	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = &serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	}
	return rest.SetKubernetesDefaults(config)
}
