package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/handloom/admin/internal/domain"
)

// resolveActorNames maps user ids to display names for a set of records.
//
// Shared by the stock ledger and the refund list, which both store an opaque
// created_by and both have to render something a person can read. One lookup per
// distinct id rather than per row: a run of movements or refunds is usually one
// admin working through it.
//
// A repo of nil resolves nothing — the storefront builds InventoryService for
// stock levels and has no business holding an admin user repository. A lookup
// that fails is logged and left empty rather than failing the read, because the
// records are worth showing without the names.
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
