-- Drop everything
DROP TABLE IF EXISTS "products";
DROP TABLE IF EXISTS "product_types";
DROP TABLE IF EXISTS "customers";
DROP TABLE IF EXISTS "orders";
DROP TABLE IF EXISTS "order_lines";


-- Create everything
CREATE TABLE "products" (
	"id" serial NOT NULL,
	"sku" varchar NOT NULL UNIQUE,
	"name" varchar NOT NULL,
	"description" TEXT NOT NULL,
	"type_id" int NOT NULL,
	"stock" int NOT NULL,
	"cost" int NOT NULL,
	"selling_price" int NOT NULL,
	PRIMARY KEY ("id"),
	FOREIGN KEY ("type_id") REFERENCES product_types("id")
);


CREATE TABLE "product_types" (
	"id" serial NOT NULL,
	"name" varchar NOT NULL UNIQUE,
	PRIMARY KEY ("id")
);


CREATE TABLE "customers" (
	"id" serial NOT NULL,
	"full_name" varchar NOT NULL,
	"company_name" varchar NOT NULL,
	"email" varchar NOT NULL,
	"address" varchar NOT NULL,
	"postal_code" varchar NOT NULL,
	"city" varchar NOT NULL,
	"country" varchar NOT NULL,
	PRIMARY KEY ("id")
);


CREATE TABLE "orders" (
	"id" INTEGER PRIMARY KEY AUTOINCREMENT,
	"customer_id" int NOT NULL,
	"created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY ("customer_id") REFERENCES customers("id")
);


CREATE TABLE "order_lines" (
	"order_id" int NOT NULL,
	"product_id" int NOT NULL,
	"amount" int NOT NULL,
	FOREIGN KEY ("order_id") REFERENCES orders("id"),
	FOREIGN KEY ("product_id") REFERENCES products("id")
);
