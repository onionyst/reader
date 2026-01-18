CREATE TABLE "categories" (
  "id" bigserial NOT NULL,
  "name" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_categories_name" ON "categories" ("name");

CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "email" text NOT NULL,
  "password" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_users_email" ON "users" ("email");

CREATE TABLE "feeds" (
  "id" bigserial NOT NULL,
  "name" text NOT NULL,
  "priority" smallint NOT NULL DEFAULT 10,
  "url" text NOT NULL,
  "website" text NOT NULL,
  "icon_url" text NOT NULL,
  "category_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_categories_feeds" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX "idx_feeds_category_priority" ON "feeds" ("category_id", "priority");

CREATE UNIQUE INDEX "uidx_feeds_url" ON "feeds" ("url");

CREATE TABLE "entries" (
  "id" bigserial NOT NULL,
  "author" text NOT NULL,
  "content" text NOT NULL,
  "date" timestamptz NOT NULL,
  "favorite" boolean NOT NULL DEFAULT false,
  "guid" text NOT NULL,
  "link" text NOT NULL,
  "read" boolean NOT NULL DEFAULT false,
  "title" text NOT NULL,
  "feed_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_feeds_entries" FOREIGN KEY ("feed_id") REFERENCES "feeds" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX "idx_entries_feed_date" ON "entries" ("feed_id", "date" DESC);

CREATE UNIQUE INDEX "uidx_entries_feed_guid" ON "entries" ("feed_id", "guid");

CREATE TABLE "tags" (
  "id" bigserial NOT NULL,
  "name" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "uidx_tags_name" ON "tags" ("name");

CREATE TABLE "entry_tags" (
  "entry_id" bigint NOT NULL,
  "tag_id" bigint NOT NULL,
  PRIMARY KEY ("entry_id", "tag_id"),
  CONSTRAINT "fk_entry_tags_entry" FOREIGN KEY ("entry_id") REFERENCES "entries" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_entry_tags_tag" FOREIGN KEY ("tag_id") REFERENCES "tags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX "idx_entry_tags_tag_entry" ON "entry_tags" ("tag_id", "entry_id");
