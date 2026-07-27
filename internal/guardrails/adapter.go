package guardrails

import (
	"context"
	"fmt"
)

// Adapter wraps Client as react.AnswerVerifier with mode semantics.
type Adapter struct {
	Mode   Mode
	Client *Client
}

// VerifyText implements react.AnswerVerifier.
func (a Adapter) VerifyText(ctx context.Context, text, retrievalContext, tenantID string) (bool, []string, error) {
	if a.Mode == ModeOff || a.Client == nil {
		return true, nil, nil
	}
	v, err := a.Client.VerifyText(ctx, text, retrievalContext, tenantID)
	if err != nil {
		if a.Mode == ModeHybrid {
			return true, nil, nil // soft-skip on transport errors
		}
		return false, nil, err
	}
	if !v.Passed {
		return false, v.Violations, nil
	}
	return true, nil, nil
}

// NewAdapter builds a verifier from mode + addr. ModeOff returns nil adapter (caller should set Engine.Verifier=nil).
func NewAdapter(mode Mode, addr string) (*Adapter, error) {
	mode = NormalizeMode(string(mode))
	if mode == ModeOff {
		return nil, nil
	}
	if addr == "" {
		return nil, fmt.Errorf("GUARDRAILS_GRPC_ADDR required for mode %s", mode)
	}
	return &Adapter{Mode: mode, Client: NewClient(addr)}, nil
}
