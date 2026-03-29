// Package ctxhelper provides utilities for securely maintaining metadata during request processing for Echo and the standard context.Context.
package ctxhelper

//go:generate go run ../../../scripts/genctxkey --name authn --type boilerplate-go/internal/usecase/boundary/auth.Authn --out .
