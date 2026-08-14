package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type sedTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewSedTool creates a new sed Tool.
func NewSedTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &sedTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *sedTool) Name() string {
	return "sed"
}

func (t *sedTool) Description() string {
	return "Stream editing and regex replacement."
}

func (t *sedTool) IsMutating() bool {
	return false
}

func (t *sedTool) Execute(ctx context.Context, args contracts_phase1.SedArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{}

	err := t.policy.Evaluate(ctx, t.Name(), []string{}, t.IsMutating())
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	limits := t.policy.Limits()

	if int64(len(args.Input)) > limits.MaxOutputBytes {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Input exceeds MaxOutputBytes", false, meta)
	}

	// Parse the expression: s/pattern/replacement/flags
	expr := args.Expression
	if !strings.HasPrefix(expr, "s/") {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Invalid sed expression format, must start with s/", false, meta)
	}

	parts := strings.Split(expr, "/")
	if len(parts) < 4 {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Invalid sed expression format", false, meta)
	}

	// Because pattern or replacement might contain escaped slashes (though we don't strictly support escaping slashes right now according to spec, let's keep it simple with Split and if there are more parts, re-join them is not trivial. Wait, the spec says "The delimiter is always /". So let's just use Split and assume 4 parts: s, pattern, replacement, flags)
    // Actually, if there are more than 4 parts, it means there are slashes in the pattern or replacement.
    // The spec says "The delimiter is always /". Let's handle it by reading from the front and back.
    if len(parts) > 4 {
        meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Invalid sed expression format, too many delimiters", false, meta)
    }

	pattern := parts[1]
	replacement := parts[2]
	flags := parts[3]

	global := strings.Contains(flags, "g")
	caseInsensitive := strings.Contains(flags, "i")

	if caseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, fmt.Sprintf("Invalid regex: %v", err), false, meta)
	}

    var result string
    var replacementsMade int

    if global {
        matches := re.FindAllString(args.Input, -1)
        replacementsMade = len(matches)
        result = re.ReplaceAllString(args.Input, replacement)
    } else {
        matches := re.FindAllString(args.Input, 1)
        replacementsMade = len(matches)
        if replacementsMade > 0 {
            // Wait, ReplaceAllString replaces *all* occurrences. We only want the first.
            // Using ReplaceAllStringFunc or manually slicing.
            loc := re.FindStringIndex(args.Input)
            if loc != nil {
                // To replace only the first occurrence and support capture groups:
                // We can use Expand or just let regexp do it on a substring? No, ReplaceAllString does all.
                // Go's regexp doesn't have a ReplaceFirstString.
                // We can use ReplaceAllString on the first match only? No.
                // The easiest way is to use ReplaceAllStringFunc? But we need to support $1, $2 etc in the replacement.
                // ReplaceAllString handles capture groups.
                // Let's manually do it:
                result = args.Input[:loc[0]] + re.ReplaceAllString(args.Input[loc[0]:loc[1]], replacement) + args.Input[loc[1]:]
            } else {
                result = args.Input
            }
        } else {
            result = args.Input
        }
    }

	if int64(len(result)) > limits.MaxOutputBytes {
		result = result[:limits.MaxOutputBytes]
		meta.Truncated = true
	}

	data := contracts_phase1.SedData{
		Result:           result,
		ReplacementsMade: replacementsMade,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}
