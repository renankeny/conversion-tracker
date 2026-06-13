package config

import (
	"log/slog"
	"time"
)

// ProjectTimezone é a única timezone usada pelo sistema. Definida como
// regra fixa do projeto (ver .kiro/steering/project-rules.md §10).
const ProjectTimezone = "America/Sao_Paulo"

// MustSetTimezone força time.Local pra ProjectTimezone. Tem que ser chamado
// no início de cada binário (cmd/api/main.go, cmd/worker/main.go) antes de
// qualquer operação que envolva time.Now().
//
// Em produção a TZ também é definida via env (TZ=America/Sao_Paulo) no
// container e o Postgres é iniciado com `-c timezone=America/Sao_Paulo`.
// Esta função é a redundância de aplicação caso a env não esteja presente.
func MustSetTimezone() {
	loc, err := time.LoadLocation(ProjectTimezone)
	if err != nil {
		slog.Error("failed to load project timezone, falling back to UTC",
			"timezone", ProjectTimezone, "err", err)
		return
	}
	time.Local = loc
}
