CREATE TABLE `addresses` (
	`id` varchar(36) NOT NULL,
	`userId` varchar(36) NOT NULL,
	`label` varchar(60) NOT NULL,
	`recipientName` varchar(120) NOT NULL,
	`recipientPhone` varchar(30) NOT NULL,
	`street` varchar(255) NOT NULL,
	`subdistrict` varchar(120),
	`district` varchar(120),
	`city` varchar(120),
	`province` varchar(120),
	`postalCode` varchar(20),
	`isPrimary` boolean NOT NULL DEFAULT false,
	`created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	`deletedAt` datetime,
	CONSTRAINT `addresses_id` PRIMARY KEY(`id`)
);
ALTER TABLE `addresses` ADD CONSTRAINT `addresses_userId_users_id_fk` FOREIGN KEY (`userId`) REFERENCES `users`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint

CREATE INDEX `idx_addresses_user_primary` ON `addresses` (`userId`,`isPrimary`);--> statement-breakpoint

## Task
- Saya sedang membuat API Gadget Online Store dengan Golang, Sqlc dan PostgresSql
-Buatkan Query, Migration, Repo, DTO, Service, Controller untuk module Address
- Function yang dibutuhkan:
Customer:
- list() //by user id
- create()
- update()
- delete()

## Rule
ganti semua nama field migration jadi snake_case, ikuti pola yang ada pada lampiran contoh agar kompatibel dengan postgres
