// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ResourceBuilder is a helper struct to build resources predefined in metadata.yaml.
// The ResourceBuilder is not thread-safe and must not to be used in multiple goroutines.
type ResourceBuilder struct {
	config ResourceAttributesConfig
	res    pcommon.Resource
}

// NewResourceBuilder creates a new ResourceBuilder. This method should be called on the start of the application.
func NewResourceBuilder(rac ResourceAttributesConfig) *ResourceBuilder {
	return &ResourceBuilder{
		config: rac,
		res:    pcommon.NewResource(),
	}
}

// SetFileName sets provided value as "file.name" attribute.
func (rb *ResourceBuilder) SetFileName(val string) {
	if rb.config.FileName.Enabled {
		rb.res.Attributes().PutStr("file.name", val)
	}
}

// SetFilePath sets provided value as "file.path" attribute.
func (rb *ResourceBuilder) SetFilePath(val string) {
	if rb.config.FilePath.Enabled {
		rb.res.Attributes().PutStr("file.path", val)
	}
}

// Emit returns the built resource and resets the internal builder state.
func (rb *ResourceBuilder) Emit() pcommon.Resource {
	r := rb.res
	rb.res = pcommon.NewResource()
	return r
}