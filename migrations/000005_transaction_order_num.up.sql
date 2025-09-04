ALTER TABLE "sufirmart"."transaction"
    RENAME COLUMN "order_id" TO "order_num";

ALTER TABLE "sufirmart"."transaction"
    ALTER COLUMN "order_num" TYPE text USING "order_num"::text;