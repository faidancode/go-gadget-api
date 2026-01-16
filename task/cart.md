## Query and Schema

CREATE TABLE `cart_items` (
	`id` varchar(36) NOT NULL,
	`cartId` varchar(36) NOT NULL,
	`product_id` varchar(36) NOT NULL,
	`quantity` int NOT NULL,
	`priceCentsAtAdd` int NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `cart_items_id` PRIMARY KEY(`id`),
	CONSTRAINT `uniq_cart_book` UNIQUE(`cartId`,`product_id`)
);
--> statement-breakpoint
CREATE TABLE `carts` (
	`id` varchar(36) NOT NULL,
	`userId` varchar(36) NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `carts_id` PRIMARY KEY(`id`),
	CONSTRAINT `carts_userId_unique` UNIQUE(`userId`)
);

ALTER TABLE `cart_items` ADD CONSTRAINT `cart_items_cartId_carts_id_fk` FOREIGN KEY (`cartId`) REFERENCES `carts`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `cart_items` ADD CONSTRAINT `cart_items_product_id_books_id_fk` FOREIGN KEY (`product_id`) REFERENCES `books`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `carts` ADD CONSTRAINT `carts_userId_users_id_fk` FOREIGN KEY (`userId`) REFERENCES `users`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint



## Task
- Saya sedang membuat API Gadget Online Store dengan Golang, Sqlc dan PostgresSql
-Buatkan Query, Schema, Repo, DTO, Service, Service Test, Controller dan Controller Test untuk module Cart
- Function yang dibutuhkan:
1. create()
2. count()
3. updateQty()
4. increment()
5. decrement()
6. deleteItem()
7. delete()
