package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Transaction holds the schema definition for the Transaction entity.
type Transaction struct {
	ent.Schema
}

// Fields of the Transaction.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique(),
		field.Float("amount").
			Comment("The amount of the transaction"),
		field.Enum("type").
			Values("INCOME", "EXPENSE").
			Default("EXPENSE").
			Comment("The type of transaction"),
		field.String("currency").
			Default("ETB").
			Comment("The currency of the transaction"),
		field.String("category").
			Comment("The category of the transaction (e.g., Food, Transport, Salary)"),
		field.String("description").
			Optional().
			Comment("Optional description of the transaction"),
		field.String("merchant").
			Optional().
			Comment("The merchant or recipient involved in the transaction"),
		field.String("emoji").
			Optional().
			Comment("An AI-generated emoji representing the category"),
		field.String("raw_input").
			Comment("The original text message sent by the user"),
		field.Time("created_at").
			Default(func() time.Time {
				return time.Now().UTC()
			}).
			Immutable(),
	}
}

// Edges of the Transaction.
func (Transaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("transactions").
			Unique().
			Required(),
	}
}
