 private async applyPaymentStatusTransition(
    order: OrderRow,
    input: UpdatePaymentStatusInput,
  ): Promise<OrderOutput> {
    const currentStatus = order.paymentStatus as PaymentStatus;
    const nextStatus = input.paymentStatus;

    if (currentStatus === nextStatus) {
      return this.getOrderDetails(order.id);
    }

    const allowed = paymentStatusTransitions[currentStatus] ?? [];
    if (!allowed.includes(nextStatus)) {
      throw new BadRequestException(
        `Cannot transition payment from ${currentStatus} to ${nextStatus}`,
      );
    }

    const now = new Date();
    const payload: Partial<typeof schema.orders.$inferInsert> = {
      paymentStatus: nextStatus,
      paymentMethod: input.paymentMethod,
      updatedAt: now,
    };

    if (nextStatus === 'PAID') {
      payload.paidAt = input.paidAt ?? order.paidAt ?? now;
      if (order.status === 'PENDING') {
        payload.status = 'PAID';
      }
    } else if (nextStatus === 'REFUNDED') {
      payload.cancelledAt = input.cancelledAt ?? order.cancelledAt ?? now;
      if (order.status === 'PENDING' || order.status === 'PAID') {
        payload.status = 'CANCELLED';
      }
    } else if (nextStatus === 'UNPAID') {
      payload.paidAt = null;
      if (order.status === 'PAID') {
        payload.status = 'PENDING';
      }
    }

    if (typeof input.note === 'string') {
      payload.note = input.note;
    }

    await this.db
      .update(schema.orders)
      .set(payload)
      .where(eq(schema.orders.id, order.id));

    return this.getOrderDetails(order.id);
  }

  async updatePaymentStatus(
    orderId: string,
    input: UpdatePaymentStatusInput,
  ): Promise<OrderOutput> {
    const order = await this.findOrderRow(orderId);
    return this.applyPaymentStatusTransition(order, input);
  }

  async updatePaymentStatusByOrderNumber(
    orderNumber: string,
    input: UpdatePaymentStatusInput,
  ): Promise<OrderOutput> {
    const order = await this.findOrderRowByOrderNumber(orderNumber);
    return this.applyPaymentStatusTransition(order, input);
  }