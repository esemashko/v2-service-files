package schema

import (
	localmixin "main/ent/schema/mixin"
	"main/hooks"
	"main/privacy/file"

	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/privacy"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// File holds the schema definition for the File entity
type File struct {
	ent.Schema
}

// Mixin of the File
func (File) Mixin() []ent.Mixin {
	return []ent.Mixin{
		localmixin.TimeMixin{},
		localmixin.LimitMixin{},
	}
}

// Policy defines the privacy policy using centralized file privacy rules
func (File) Policy() ent.Policy {
	return privacy.Policy{
		Query: privacy.QueryPolicy{
			file.QueryRule(),
		},
		Mutation: privacy.MutationPolicy{
			file.MutationRule(),
		},
	}
}

// Hooks of the File
func (File) Hooks() []ent.Hook {
	return []ent.Hook{
		// Автоматически удаляет файл из S3 при удалении записи из БД
		hooks.WithFileS3Deletion(),
	}
}

func (File) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("original_name").
			NotEmpty().
			Comment("Оригинальное имя загруженного файла"),
		field.String("storage_key").
			NotEmpty().
			Comment("Уникальный ключ в хранилище S3"),
		field.String("mime_type").
			NotEmpty().
			Comment("MIME-тип файла"),
		field.Int64("size").
			Positive().
			Comment("Размер файла в байтах"),
		field.String("path").
			Optional().
			Comment("Путь к файлу в хранилище (deprecated, используется storage_key)"),
		field.String("description").
			Optional().
			Comment("Описание файла"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Дополнительные метаданные файла"),
	}
}

func (File) Edges() []ent.Edge {
	return []ent.Edge{
		// Пользователь, загрузивший файл
		edge.To("uploader", User.Type).
			Unique().
			Required(),

		// Связь с тикетами через промежуточную таблицу
		edge.From("ticket_files", TicketFile.Type).
			Ref("file"),

		// Связь с комментариями через промежуточную таблицу
		edge.From("comment_files", TicketCommentFile.Type).
			Ref("file"),

		// Связь с сообщениями, где используется файл
		edge.From("messages", Message.Type).
			Ref("files").
			Comment("Messages that have this file attached"),
	}
}

func (File) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("storage_key").
			Unique(),
	}
}

// Annotations defines GraphQL and database annotations
func (File) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "files"},
		entgql.RelayConnection(),
		entgql.QueryField(),
		entgql.Mutations(
			entgql.MutationCreate(),
			entgql.MutationUpdate(),
		),
		entgql.MultiOrder(),
		entgql.OrderField("CREATE_TIME"),
	}
}
