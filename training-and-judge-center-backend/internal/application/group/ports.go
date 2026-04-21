package group

import "context"

// UserDisplay es el dato mínimo para mostrar usuarios (leads) en la respuesta.
type UserDisplay struct {
	Nickname string
	Name     string
}

// UserProvider consulta la tabla users sin exponer su dominio a groups.
type UserProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}

// PreferencesReader lee users.preferences (JSONB) y expone las flags que
// necesita el dominio de grupos.
type PreferencesReader interface {
	HideGlobalGroup(ctx context.Context, userID string) (bool, error)
}
