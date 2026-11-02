CREATE TABLE applications (
 id text PRIMARY KEY,
 request_key text UNIQUE NOT NULL,
 payload jsonb NOT NULL,
 host text NOT NULL,
 port integer NOT NULL CHECK (port BETWEEN 1 AND 65535),
 token text NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
 reason text NOT NULL DEFAULT '',
 server_id integer REFERENCES servers(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 decided_at timestamptz,
 notified_at timestamptz,
 next_attempt timestamptz NOT NULL DEFAULT now(),
 attempts integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX applications_pending_address ON applications(lower(host), port) WHERE status='pending';
CREATE INDEX applications_delivery ON applications(next_attempt) WHERE notified_at IS NULL;
CREATE TABLE application_limits (client text PRIMARY KEY, window_start timestamptz NOT NULL, count integer NOT NULL);
