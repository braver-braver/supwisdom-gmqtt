package logging

import (
	"fmt"

	"go.uber.org/zap"
)

const maxErrorDepth = 32

// WithCaller skips one wrapper frame so the caller points to the business code.
func WithCaller(logger *zap.Logger) *zap.Logger {
	return logger.WithOptions(zap.AddCallerSkip(1))
}

// Scene attaches a stable module/op pair and optional extra fields.
func Scene(module string, op string, fields ...zap.Field) []zap.Field {
	scene := make([]zap.Field, 0, len(fields)+2)
	if module != "" {
		scene = append(scene, zap.String("module", module))
	}
	if op != "" {
		scene = append(scene, zap.String("op", op))
	}
	scene = append(scene, fields...)
	return scene
}

// Err attaches the error itself together with the unwrapped error chain.
func Err(err error) []zap.Field {
	if err == nil {
		return nil
	}
	fields := []zap.Field{
		zap.Error(err),
		zap.String("error_type", fmt.Sprintf("%T", err)),
	}
	if chain := ErrorChain(err); len(chain) != 0 {
		fields = append(fields, zap.Strings("error_chain", chain))
	}
	return fields
}

// ErrorChain returns the error text for each unwrap level.
func ErrorChain(err error) []string {
	if err == nil {
		return nil
	}
	chain := make([]string, 0, 4)
	for depth := 0; err != nil && depth < maxErrorDepth; depth++ {
		chain = append(chain, err.Error())
		next := unwrap(err)
		if next == err {
			break
		}
		err = next
	}
	return chain
}

type singleUnwrapper interface {
	Unwrap() error
}

type multiUnwrapper interface {
	Unwrap() []error
}

func unwrap(err error) error {
	if err == nil {
		return nil
	}
	if multi, ok := err.(multiUnwrapper); ok {
		errs := multi.Unwrap()
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}
	if single, ok := err.(singleUnwrapper); ok {
		return single.Unwrap()
	}
	return nil
}
