CREATE TABLE `order_items` (
	`id` varchar(36) NOT NULL,
	`orderId` varchar(36) NOT NULL,
	`bookId` varchar(36) NOT NULL,
	`titleSnapshot` varchar(200) NOT NULL,
	`unitPriceCents` int NOT NULL,
	`quantity` int NOT NULL,
	`totalCents` int NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `order_items_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE TABLE `orders` (
	`id` varchar(36) NOT NULL,
	`orderNumber` varchar(32),
	`userId` varchar(36) NOT NULL,
	`status` varchar(16) NOT NULL DEFAULT 'PENDING',
	`paymentMethod` varchar(32),
	`paymentStatus` varchar(16) NOT NULL DEFAULT 'UNPAID',
	`addressSnapshot` json NOT NULL,
	`subtotalCents` int NOT NULL,
	`discountCents` int NOT NULL DEFAULT 0,
	`shippingCents` int NOT NULL DEFAULT 0,
	`totalCents` int NOT NULL,
	`note` varchar(255),
	`placedAt` datetime NOT NULL,
	`paidAt` datetime,
	`cancelledAt` datetime,
	`cancelReason` varchar(100),
	`completedAt` datetime,
	`receipt_no` varchar(50),
	`midtransOrderId` varchar(50) NOT NULL,
	`snapToken` varchar(255),
	`snapRedirectUrl` varchar(255),
	`snapTokenExpiredAt` datetime,
	`created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	`deletedAt` datetime,
	CONSTRAINT `orders_id` PRIMARY KEY(`id`),
	CONSTRAINT `orders_orderNumber_unique` UNIQUE(`orderNumber`),
	CONSTRAINT `orders_receipt_no_unique` UNIQUE(`receipt_no`)
);

ALTER TABLE `order_items` ADD CONSTRAINT `order_items_orderId_orders_id_fk` FOREIGN KEY (`orderId`) REFERENCES `orders`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `order_items` ADD CONSTRAINT `order_items_bookId_books_id_fk` FOREIGN KEY (`bookId`) REFERENCES `books`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE `orders` ADD CONSTRAINT `orders_userId_users_id_fk` FOREIGN KEY (`userId`) REFERENCES `users`(`id`) ON DELETE no action ON UPDATE no action;--> statement-breakpoint

CREATE INDEX `idx_order_items_order` ON `order_items` (`orderId`);--> statement-breakpoint
CREATE INDEX `idx_orders_user_status` ON `orders` (`userId`,`status`);--> statement-breakpoint
CREATE INDEX `idx_orders_placedAt` ON `orders` (`placedAt`);--> statement-breakpoint


## Task
- Saya sedang membuat API Gadget Online Store dengan Golang, Sqlc dan PostgresSql
-Buatkan Query, Schema, Repo, DTO, Service, Service Test, Controller dan Controller Test untuk module Order
- Function yang dibutuhkan:
Customer:
- checkout()
- list()
- detail()
- cancel()
- update() // ubah status dari DELIVERED jadi COMPLETED

Admin
- list()
- detail()
- update() // ubah SHIPPING jadi DELIVERED, sementara sampai nanti ada integrasi dengan third party shipment

## Rule
ganti semua nama field jadi snake_case, ikuti pola yang ada pada lampiran contoh agar kompatibel dengan postgres
