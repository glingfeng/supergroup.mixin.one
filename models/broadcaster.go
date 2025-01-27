package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/supergroup.mixin.one/session"
)

const broadcasters_DDL = `
CREATE TABLE IF NOT EXISTS broadcasters (
	user_id	          VARCHAR(36) PRIMARY KEY CHECK (user_id ~* '^[0-9a-f-]{36,36}$'),
	created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS broadcasters_updatedx ON broadcasters(updated_at);
`

var broadcasterColumns = []string{"user_id", "created_at", "updated_at"}

func (b *Broadcaster) values() []interface{} {
	return []interface{}{b.UserId, b.CreatedAt, b.UpdatedAt}
}

type Broadcaster struct {
	UserId    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (current *User) CreateBroadcaster(ctx context.Context, identity int64) (*User, error) {
	if !current.isAdmin() {
		return nil, session.ForbiddenError(ctx)
	}

	users, err := findUsersByIdentityNumber(ctx, identity)
	if err != nil {
		return nil, err
	} else if len(users) == 0 {
		return nil, session.BadDataError(ctx)
	}
	user := users[0]

	query := fmt.Sprintf("INSERT INTO broadcasters(user_id,created_at,updated_at) VALUES ($1,$2,$3) ON CONFLICT (user_id) DO UPDATE SET updated_at=EXCLUDED.updated_at")
	_, err = session.Database(ctx).ExecContext(ctx, query, user.UserId, time.Now(), time.Now())
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return user, nil
}

func ReadBroadcasters(ctx context.Context) ([]*User, error) {
	query := fmt.Sprintf("SELECT %s FROM users WHERE user_id IN (SELECT user_id FROM broadcasters ORDER BY updated_at DESC LIMIT 5)", strings.Join(usersCols, ","))
	rows, err := session.Database(ctx).QueryContext(ctx, query)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := userFromRow(rows)
		if err != nil {
			return users, session.TransactionError(ctx, err)
		}
		users = append(users, u)
	}
	return users, nil
}
