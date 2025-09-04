CREATE TABLE IF NOT EXISTS "sufirmart"."order" (
  "user_id" uuid NOT NULL,
  "order_num" text NOT NULL,
  "status" smallint NOT NULL,
  "uploaded_at" timestamp(0) with time zone NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("user_id", "order_num"),
  CONSTRAINT "order_user_id" FOREIGN KEY ("user_id") REFERENCES "sufirmart"."user" ("id"),
  CONSTRAINT "order_num_unique" UNIQUE ("order_num")
);
