package emitterir

import "k8s.io/apimachinery/pkg/types"

// ExtensionFeatureKey is a unique identifier for an extension feature type.
type ExtensionFeatureKey string

// ExtensionFeatureIR represents an intermediate representation of an extension feature
// that can be attached to Gateway API resources (e.g., HTTPRoute rules, Gateway listeners).
// Implementations track parsing state, errors, and source information for the feature.
type ExtensionFeatureIR interface {
	// IsParsed returns true if this extension feature has been parsed by an emitter.
	IsParsed() bool
	// SetParsed marks this extension feature as successfully parsed by an emitter.
	// Emitter MUST call this method upon successful parsing.
	SetParsed()
	// GetError returns any error that occurred during parsing or conversion.
	GetError() error
	// SetError sets an error that occurred during parsing or conversion.
	// Emitter MUST call this method if an error occurs.
	SetError(err error)
	// GetSource returns the source information (provider and annotation key) for this extension feature.
	GetSource() *ExtensionFeatureSource
	// SetSource sets the source information (provider and annotation key) for this extension feature.
	// Provider MUST call this method when creating the extension feature IR.
	SetSource(source *ExtensionFeatureSource)

	// Equals compares this extension feature with another to determine if they are identical.
	// This method is used to merge IR among multiple attach resources (e.g., xRoute rule, Gateway listener).
	Equals(other ExtensionFeatureIR) bool
}

type ExtensionFeatureSource struct {
	IngressNN     types.NamespacedName
	AnnotationKey string
}

type ExtensionFeatureMetadata struct {
	parsed bool
	error  error
	source *ExtensionFeatureSource
}

func (m *ExtensionFeatureMetadata) SetParsed() {
	m.parsed = true
}

func (m *ExtensionFeatureMetadata) IsParsed() bool {
	return m.parsed
}

func (m *ExtensionFeatureMetadata) GetError() error {
	return m.error
}

func (m *ExtensionFeatureMetadata) SetError(err error) {
	m.error = err
}

func (m *ExtensionFeatureMetadata) GetSource() *ExtensionFeatureSource {
	return m.source
}

func (m *ExtensionFeatureMetadata) SetSource(source *ExtensionFeatureSource) {
	m.source = source
}
