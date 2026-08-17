CREATE TABLE IF NOT EXISTS checklist_templates (
  template_id text NOT NULL,
  version integer NOT NULL CHECK (version > 0),
  published boolean NOT NULL DEFAULT false,
  payload jsonb NOT NULL,
  PRIMARY KEY (template_id, version)
);

CREATE TABLE IF NOT EXISTS releases (
  id text PRIMARY KEY,
  application_id text NOT NULL,
  environment_id text NOT NULL,
  state text NOT NULL,
  revision bigint NOT NULL CHECK (revision > 0),
  idempotency_key text NOT NULL UNIQUE,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS releases_state_updated_idx ON releases(state, updated_at DESC);
CREATE INDEX IF NOT EXISTS releases_application_idx ON releases(application_id, environment_id);

CREATE TABLE IF NOT EXISTS audit_events (
  id text PRIMARY KEY,
  release_id text NOT NULL REFERENCES releases(id) ON DELETE RESTRICT,
  actor_id text NOT NULL,
  action text NOT NULL,
  detail text NOT NULL,
  previous_hash text NOT NULL,
  hash text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS audit_release_time_idx ON audit_events(release_id, created_at, id);

CREATE TABLE IF NOT EXISTS deployment_executions (
  id text PRIMARY KEY,
  release_id text NOT NULL REFERENCES releases(id) ON DELETE RESTRICT,
  status text NOT NULL,
  payload jsonb NOT NULL,
  started_at timestamptz NOT NULL,
  finished_at timestamptz
);
