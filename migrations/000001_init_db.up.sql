CREATE SCHEMA IF NOT EXISTS "sufirmart";

CREATE TABLE IF NOT EXISTS "sufirmart"."user" (
  "id" uuid NOT NULL,
  "login" text NOT NULL,
  "password" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "sufirmart"."auth" (
  "user_id" uuid NOT NULL references "sufirmart"."user",
  "token" text NOT NULL,
  "expired_at" timestamp(0) with time zone NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);
