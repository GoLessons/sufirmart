CREATE TABLE IF NOT EXISTS "sufirmart"."transaction" (
  id uuid NOT NULL,
  user_id uuid NOT NULL,
  order_id uuid NOT NULL,
  accrual numeric(16, 2) NOT NULL DEFAULT 0::NUMERIC,
  withdraw numeric(16, 2) NOT NULL DEFAULT 0::NUMERIC,
  created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  processed_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  status smallint NOT NULL,
  comment text,
  PRIMARY KEY ("id"),
  CONSTRAINT "transaction_user_id" FOREIGN KEY ("user_id") REFERENCES "sufirmart"."user" ("id")
);
