package identity

import "context"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleController Role = "controller"
	RoleReviewer   Role = "reviewer"
)

type Principal struct {
	UserID           string `json:"userId"`
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	Email            string `json:"email"`
	DisplayName      string `json:"displayName"`
	Role             Role   `json:"role"`
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
