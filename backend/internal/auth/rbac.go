package auth

import "context"

// CanEditPixels verifica se o ator autenticado pode criar/editar pixels e links rastreáveis
// dentro do escopo estrito da sua própria empresa.
func CanEditPixels(ctx context.Context) bool {
	c, ok := ClaimsFrom(ctx)
	if !ok || c == nil {
		return false
	}
	if c.IsAPIKey() {
		return c.AccountID > 0 && c.HasAPIScope("pixels", true)
	}
	if c.IsOwner() || c.Role == "admin" || c.Role == "manager" {
		return true
	}
	if c.Role == "employee" {
		canEdit, _ := c.Permissions["edit_pixels"].(bool)
		return canEdit
	}
	return false
}

// CanManageUsers verifica se o usuário pode listar/criar/editar subordinados no painel.
func CanManageUsers(ctx context.Context) bool {
	c, ok := ClaimsFrom(ctx)
	if !ok || c == nil || c.IsAPIKey() {
		return false
	}
	return c.IsOwner() || c.Role == "admin" || c.Role == "manager"
}
