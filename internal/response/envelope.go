package response

import (
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
)

type builder struct{}

// NewBuilder creates a new EnvelopeBuilder.
func NewBuilder() contracts.EnvelopeBuilder {
	return &builder{}
}

// Success builds an ok=true envelope with the given tool-specific data.
func (b *builder) Success(tool string, data interface{}, meta contracts.Meta) contracts.Envelope {
	return contracts.Envelope{
		Version: "1",
		OK:      true,
		Tool:    tool,
		Data:    data,
		Meta:    meta,
		Error:   nil,
	}
}

// Failure builds an ok=false envelope with a structured error.
func (b *builder) Failure(tool string, code contracts.ErrorCode, message string, retryable bool, meta contracts.Meta) contracts.Envelope {
	return contracts.Envelope{
		Version: "1",
		OK:      false,
		Tool:    tool,
		Data:    nil,
		Meta:    meta,
		Error: &contracts.Error{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
	}
}
