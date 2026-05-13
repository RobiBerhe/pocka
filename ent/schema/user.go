package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique(),
		field.Int64("telegram_id").
			Unique().
			Comment("The Telegram User ID"),
		field.String("username").
			Optional().
			Comment("The Telegram Username"),
		field.String("timezone").
			Default("Africa/Addis_Ababa").
			Comment("User's timezone for reports"),
		field.String("currency").
			Default("ETB").
			Comment("User's default currency"),
		field.String("language").
			Default("en").
			Comment("User's preferred language"),
		field.Bool("is_premium").
			Default(false),
		field.Int("current_streak").
			Default(0),
		field.UUID("referred_by", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("The ID of the user who referred this user"),
		field.Time("created_at").
			Default(func() time.Time {
				return time.Now().UTC()
			}).
			Immutable(),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", Transaction.Type),
	}
}
