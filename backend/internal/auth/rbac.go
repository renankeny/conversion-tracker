package auth

import "context"

// CanEditPixels verifica se o usuário autenticado tem permissão humana para editar pixels.
// Chaves de API S2S nunca podem mutar configurações de pixels.
func CanEditPixels(ctx context.Context) bool {
	c, ok := ClaimsFrom(ctx)
	if !ok || c == nil || c.IsAPIKey() {
		return false
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
