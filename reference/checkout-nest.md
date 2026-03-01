 async checkout(
    input: CheckoutOrderInput,
    options?: { idempotencyKey?: string },
  ): Promise<CheckoutResult> {
    const trimmedKey = options?.idempotencyKey?.trim();
    console.log(trimmedKey);
    const cacheKey =
      trimmedKey && trimmedKey.length > 0
        ? `${input.userId}:${trimmedKey}`
        : null;

    if (cacheKey) {
      const existing = this.idempotencyCache.get(cacheKey);
      if (existing) {
        return {
          order: await this.getOrderDetails(existing, input.userId),
          payment: null,
        };
      }

      // ⛔ LOCK sebelum transaction
      this.idempotencyCache.set(cacheKey, 'LOCKED');
    }

    const now = new Date();
    const orderId = randomUUID();
    const orderNumber = this.generateOrderNumber();

    const customerProfile = await this.loadCustomerProfile(input.userId);
    let paymentPayload: CheckoutPaymentPayload | null = null;

    await this.db.transaction(async (tx) => {
      const cart = await this.fetchCartWithItems(tx, input.userId);
      console.log('CART');
      console.log({ cart });
      const [address] = await tx
        .select()
        .from(schema.addresses)
        .where(
          and(
            eq(schema.addresses.id, input.addressId),
            eq(schema.addresses.userId, input.userId),
            sql`${schema.addresses.deletedAt} IS NULL`,
          ),
        )
        .limit(1);

      if (!address) {
        throw new BadRequestException('Address not found for user');
      }

      const addressSnapshot = this.buildAddressSnapshot(address);
      let subtotalCents = 0;

      const stockAdjustments: Array<{
        bookId: string;
        nextStock: number;
      }> = [];
      const orderItemsPayload: (typeof schema.orderItems.$inferInsert)[] = [];

      for (const item of cart.items) {
        if (item.bookStock < item.quantity) {
          throw new BadRequestException(
            `Insufficient stock for ${item.bookTitle}`,
          );
        }

        const totalItemCents = item.priceCentsAtAdd * item.quantity;
        subtotalCents += totalItemCents;

        stockAdjustments.push({
          bookId: item.bookId,
          nextStock: item.bookStock - item.quantity,
        });

        orderItemsPayload.push({
          id: randomUUID(),
          orderId,
          bookId: item.bookId,
          titleSnapshot: item.bookTitle,
          unitPriceCents: item.priceCentsAtAdd,
          quantity: item.quantity,
          totalCents: totalItemCents,
        });
      }

      const discountCents = Math.max(0, input.discountCents ?? 0);
      const shippingCents = Math.max(0, input.shippingCents ?? 0);
      const totalCents = Math.max(
        0,
        subtotalCents - discountCents + shippingCents,
      );
      const midtransItems = orderItemsPayload.map((item) => ({
        id: item.bookId,
        price: item.unitPriceCents,
        quantity: item.quantity,
        name: item.titleSnapshot.slice(0, 50),
      }));
      if (shippingCents > 0) {
        midtransItems.push({
          id: 'shipping-fee',
          price: shippingCents,
          quantity: 1,
          name: 'Shipping Fee',
        });
      }
      if (discountCents > 0) {
        midtransItems.push({
          id: 'discount',
          price: -discountCents,
          quantity: 1,
          name: 'Discount',
        });
      }

      const initialStatus: OrderStatus =
        (input.initialStatus as OrderStatus) ?? 'PENDING';
      const paymentStatus =
        initialStatus === 'PAID'
          ? ('PAID' as OrderOutput['paymentStatus'])
          : ('UNPAID' as OrderOutput['paymentStatus']);

      if (this.midtransService) {
        const addressName = this.splitFullName(addressSnapshot.recipientName);
        paymentPayload = await this.midtransService.createTransactionToken({
          orderId: orderNumber,
          grossAmount: totalCents,
          customer: {
            firstName:
              customerProfile.firstName ?? addressName.firstName ?? undefined,
            lastName:
              customerProfile.lastName ?? addressName.lastName ?? undefined,
            email: customerProfile.email ?? undefined,
            phone:
              customerProfile.phone ??
              addressSnapshot.recipientPhone ??
              undefined,
          },
          items: midtransItems,
        });
      }
      console.log({ paymentPayload });

      await tx.insert(schema.orders).values({
        id: orderId,
        orderNumber,
        userId: input.userId,
        status: initialStatus,
        paymentMethod: input.paymentMethod,
        paymentStatus,
        addressSnapshot,
        subtotalCents,
        discountCents,
        shippingCents,
        totalCents,
        note: input.note ?? null,
        placedAt: now,
        paidAt: initialStatus === 'PAID' ? now : null,
        midtransOrderId: orderId,
        snapToken: paymentPayload?.snapToken ?? null,
        snapRedirectUrl: paymentPayload?.redirectUrl ?? null,
        snapTokenExpiredAt: paymentPayload ? addHours(now, 24) : null,
      });

      await tx.insert(schema.orderItems).values(orderItemsPayload);
      for (const adj of stockAdjustments) {
        await tx
          .update(schema.books)
          .set({
            stock: adj.nextStock,
            updatedAt: new Date(),
          })
          .where(eq(schema.books.id, adj.bookId));
      }

      await tx
        .delete(schema.cartItems)
        .where(eq(schema.cartItems.cartId, cart.cartId));
      await tx
        .update(schema.carts)
        .set({ updatedAt: new Date() })
        .where(eq(schema.carts.id, cart.cartId));
    });

    const order = await this.getOrderDetails(orderId, input.userId);
    if (cacheKey) {
      this.idempotencyCache.set(cacheKey, order.id);
    }

    if (this.paymentIntegration) {
      await this.paymentIntegration.handleAfterCheckout(order, {
        idempotencyKey: trimmedKey ?? undefined,
      });
    }

    return { order, payment: paymentPayload };
  }