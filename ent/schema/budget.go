package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Budget holds the schema definition for the Budget entity.
type Budget struct {
	ent.Schema
}

// Fields of the Budget.
func (Budget) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique(),
		field.Float("amount").
			Comment("The total budget amount"),
		field.String("category").
			Optional().
			Comment("The category this budget applies to. If empty, it's the total monthly budget."),
		field.String("period").
			Default("monthly").
			Comment("The period of the budget: daily, weekly, monthly, yearly"),
		field.Time("created_at").
			Default(func() time.Time {
				return time.Now().UTC()
			}).
			Immutable(),
	}
}

// Edges of the Budget.
func (Budget) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("budgets").
			Unique().
			Required(),
	}
}
