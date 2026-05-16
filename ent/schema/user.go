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
		field.String("full_name").
			Optional().
			Comment("User's preferred full name"),
		field.String("phone_number").
			Optional().
			Comment("User's phone number"),
		field.String("onboarding_state").
			Default("AWAITING_NAME").
			Comment("Current step in the onboarding flow"),
		field.Bool("onboarding_completed").
			Default(false).
			Comment("Whether the user has completed onboarding"),
		field.Time("last_log_date").
			Optional().
			Comment("The date of the last transaction in user's local timezone"),
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
