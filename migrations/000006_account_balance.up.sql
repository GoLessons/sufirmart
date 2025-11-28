CREATE TABLE IF NOT EXISTS "sufirmart"."account" (
    user_id uuid NOT NULL,
    current_balance numeric(16, 2) NOT NULL DEFAULT 0::NUMERIC,
    PRIMARY KEY ("user_id"),
    CONSTRAINT "account_user_id" FOREIGN KEY ("user_id") REFERENCES "sufirmart"."user" ("id")
);