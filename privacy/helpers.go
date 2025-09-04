package privacy

import (
	"context"

	"entgo.io/ent/privacy"
	federation "github.com/esemashko/v2-federation"
	"github.com/google/uuid"
)

// ValidateAuth проверяет базовую аутентификацию и активность пользователя
// Возвращает userID или ошибку
func ValidateAuth(ctx context.Context) (uuid.UUID, error) {
	userIDPtr := federation.GetUserID(ctx)
	if userIDPtr == nil || *userIDPtr == uuid.Nil {
		return uuid.Nil, privacy.Denyf(ErrAuthenticationRequired)
	}
	userID := *userIDPtr

	return userID, nil
}
