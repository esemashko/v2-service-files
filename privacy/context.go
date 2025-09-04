package privacy

import (
	"context"

	"entgo.io/ent/privacy"
)

// WithSystemContext создает контекст для системных операций (cron jobs, migrations).
// Все privacy правила будут пропущены.
func WithSystemContext(ctx context.Context) context.Context {
	return privacy.DecisionContext(ctx, privacy.Allow)
}
