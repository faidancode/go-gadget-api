## Query and Schema

CREATE TABLE `wishlist_items` (
	`id` varchar(36) NOT NULL,
	`wishlistId` varchar(36) NOT NULL,
	`productId` varchar(36) NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `wishlist_items_id` PRIMARY KEY(`id`),
	CONSTRAINT `uniq_wishlist_product` UNIQUE(`wishlistId`,`productId`)
);
--> statement-breakpoint
CREATE TABLE `wishlists` (
	`id` varchar(36) NOT NULL,
	`userId` varchar(36) NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `wishlists_id` PRIMARY KEY(`id`),
	CONSTRAINT `wishlists_userId_unique` UNIQUE(`userId`)
);

ALTER TABLE `wishlist_items` ADD CONSTRAINT `wishlist_items_wishlistId_wishlists_id_fk` FOREIGN KEY (`wishlistId`) REFERENCES `wishlists`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `wishlist_items` ADD CONSTRAINT `wishlist_items_productId_products_id_fk` FOREIGN KEY (`productId`) REFERENCES `products`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `wishlists` ADD CONSTRAINT `wishlists_userId_users_id_fk` FOREIGN KEY (`userId`) REFERENCES `users`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint


## Task
- Saya sedang membuat API Gadget Online Store dengan Golang, Sqlc dan PostgresSql
-Buatkan Query, Schema, Repo, DTO, Service, Service Test, Controller dan Controller Test untuk module Cart
- Function yang dibutuhkan:
1. create()
2. list()
3. delete()

- Buatkan Service Test dan Controller Test
1. create success and failed
2. list success and failed
3. delete success and failed

## Rule
- ganti semua nama field jadi snake_case, ikuti pola yang ada pada lampiran contoh agar kompatibel dengan postgres
- Ikuti semua pola yang ada pada referensi, jangan membuat pola baru
