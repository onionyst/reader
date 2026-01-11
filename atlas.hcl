locals {
  pg_user = getenv("POSTGRES_USER")
  pg_pass = urlescape(getenv("POSTGRES_PASSWORD"))
  pg_host = getenv("POSTGRES_HOST")
  pg_port = getenv("POSTGRES_PORT")
  pg_db   = getenv("POSTGRES_DB")

  db_url = "postgres://${local.pg_user}:${local.pg_pass}@${local.pg_host}:${local.pg_port}/${local.pg_db}?search_path=public&sslmode=disable"
}

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/loader",
  ]
}

env "develop" {
  src = data.external_schema.gorm.url
  dev = "docker+postgres://_/postgres:18.1-alpine/dev?search_path=public"
  migration {
    dir = "file://db/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "deploy" {
  url = local.db_url
  migration {
    dir = "file:///migrations"
  }
}
