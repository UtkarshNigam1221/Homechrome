package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
)

// resolveActorNames maps user ids to names, one lookup per distinct id. A nil repo
// resolves nothing, and a failed lookup is left empty rather than failing the read.
func resolveActorNames(ctx context.Context, repo domain.UserRepository, ids []string) map[string]string {
	names := make(map[string]string, len(ids))
	if repo == nil {
		return names
	}

	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, seen := names[id]; seen {
			continue
		}

		var name string
		user, err := repo.GetByID(ctx, id)
		if err != nil {
			slog.WarnContext(ctx, "Failed to resolve actor name", "user_id", id, "error", err)
		} else if user != nil {
			name = strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name == "" {
				// The seeded accounts carry no first or last name.
				name = user.Email
			}
		}
		names[id] = name
	}

	return names
}
