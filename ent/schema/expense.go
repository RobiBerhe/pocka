package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Expense holds the schema definition for the Expense entity.
type Expense struct {
	ent.Schema
}

// Fields of the Expense.
func (Expense) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique(),
		field.Float("amount").
			Comment("The amount spent"),
		field.String("currency").
			Default("ETB").
			Comment("The currency of the expense"),
		field.String("category").
			Comment("The category of the expense (e.g., Food, Transport)"),
		field.String("raw_input").
			Comment("The original text message sent by the user"),
		field.Time("created_at").
			Default(func() time.Time {
				return time.Now().UTC()
			}).
			Immutable(),
	}
}

// Edges of the Expense.
func (Expense) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("expenses").
			Unique().
			Required(),
	}
}
