// Package ctxhelper provides utilities for securely maintaining metadata during request processing for Echo and the standard context.Context.
package ctxhelper

//go:generate go run ../../../scripts/genctxkey --name Authn --type "auth.Authn" --import boilerplate-go/internal/usecase/boundary/auth --out .
