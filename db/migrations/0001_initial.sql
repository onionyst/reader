CREATE TABLE "categories" (
  "id" bigserial NOT NULL,
  "name" TEXT NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_categories_name" ON "categories" ("name");

CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "email" TEXT NOT NULL,
  "password" TEXT NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_users_email" ON "users" ("email");

CREATE TABLE "feeds" (
  "id" bigserial NOT NULL,
  "name" TEXT NOT NULL,
  "priority" SMALLINT NOT NULL DEFAULT 10,
  "url" TEXT NOT NULL,
  "website" TEXT NOT NULL,
  "icon_url" TEXT NOT NULL,
  "category_id" BIGINT NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_categories_feeds" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX "idx_feeds_category_id" ON "feeds" ("category_id");

CREATE UNIQUE INDEX "uidx_feeds_url" ON "feeds" ("url");

CREATE TABLE "entries" (
  "id" bigserial NOT NULL,
  "author" TEXT NOT NULL,
  "content" TEXT NOT NULL,
  "date" TIMESTAMPTZ NOT NULL,
  "favorite" BOOLEAN NOT NULL DEFAULT FALSE,
  "guid" TEXT NOT NULL,
  "link" TEXT NOT NULL,
  "read" BOOLEAN NOT NULL DEFAULT FALSE,
  "title" TEXT NOT NULL,
  "feed_id" BIGINT NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_feeds_entries" FOREIGN KEY ("feed_id") REFERENCES "feeds" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "uidx_entries_feed_guid" ON "entries" ("feed_id", "guid");

CREATE TABLE "tags" (
  "id" bigserial NOT NULL,
  "name" TEXT NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_tags_name" ON "tags" ("name");

CREATE TABLE "entry_tags" (
  "entry_id" BIGINT NOT NULL,
  "tag_id" BIGINT NOT NULL,
  PRIMARY KEY ("entry_id", "tag_id"),
  CONSTRAINT "fk_entry_tags_entry" FOREIGN KEY ("entry_id") REFERENCES "entries" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_entry_tags_tag" FOREIGN KEY ("tag_id") REFERENCES "tags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX "idx_entry_tags_tag_entry" ON "entry_tags" ("tag_id", "entry_id");
