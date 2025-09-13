-- Create "files" table
CREATE TABLE "files" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "create_time" timestamptz NOT NULL,
  "update_time" timestamptz NOT NULL,
  "created_by" uuid NOT NULL,
  "original_name" character varying NOT NULL,
  "storage_key" character varying NOT NULL,
  "mime_type" character varying NOT NULL,
  "size" bigint NOT NULL,
  "path" character varying NULL,
  "description" character varying NULL,
  "metadata" jsonb NULL,
  PRIMARY KEY ("id")
);
-- Create index "file_storage_key" to table: "files"
CREATE UNIQUE INDEX "file_storage_key" ON "files" ("storage_key");
