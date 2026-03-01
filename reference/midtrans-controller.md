import {
  Body,
  BadRequestException,
  Controller,
  ForbiddenException,
  HttpCode,
  HttpStatus,
  Post,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { createHash } from 'crypto';
import { OrdersService } from './orders.service';
import {
  MidtransNotificationInput,
  MidtransNotificationSchema,
} from './schemas/orders.schemas';

@Controller('v1/midtrans')
export class OrdersMidtransController {
  constructor(
    private readonly ordersService: OrdersService,
    private readonly configService: ConfigService,
  ) {}

  private verifySignature(payload: MidtransNotificationInput) {
    const serverKey = this.configService.get<string>('MIDTRANS_SERVER_KEY');
    if (!serverKey) {
      throw new BadRequestException('Midtrans server key is not configured');
    }

    const expected = createHash('sha512')
      .update(
        `${payload.order_id}${payload.status_code}${payload.gross_amount}${serverKey}`,
      )
      .digest('hex');

    const incomingSignature = payload.signature_key.trim().toLowerCase();
    if (expected !== incomingSignature) {
      throw new ForbiddenException('Invalid Midtrans signature');
    }
  }

  private parseGrossAmount(amount: string) {
    const parsed = Number.parseFloat(amount);
    if (Number.isNaN(parsed)) {
      throw new BadRequestException('Invalid gross amount');
    }
    return Math.round(parsed);
  }

  @Post('notification')
  @HttpCode(HttpStatus.OK)
  async handleNotification(@Body() body: unknown) {
    const payload = MidtransNotificationSchema.parse(body);
    this.verifySignature(payload);
    console.log({ payload });

    const summary = await this.ordersService.getOrderSummaryByOrderNumber(
      payload.order_id,
    );

    const shouldMarkPaid =
      payload.transaction_status === 'settlement' ||
      (payload.transaction_status === 'capture' &&
        payload.fraud_status === 'accept');

    if (!shouldMarkPaid) {
      return { success: true };
    }

    if (payload.transaction_status === 'expire') {
      await this.ordersService.cancelOrderBySystem(payload.order_id);
    }

    const amountCents = this.parseGrossAmount(payload.gross_amount);

    const expectedGross = Math.max(
      0,
      summary.subtotalCents - summary.discountCents + summary.shippingCents,
    );

    if (amountCents !== expectedGross) {
      throw new BadRequestException('Gross amount does not match order total');
    }

    const paidAt = payload.transaction_time
      ? new Date(payload.transaction_time)
      : new Date();

    if (Number.isNaN(paidAt.getTime())) {
      throw new BadRequestException('Invalid transaction time');
    }

    await this.ordersService.updatePaymentStatus(summary.orderId, {
      paymentStatus: 'PAID',
      paymentMethod: payload.payment_type,
      paidAt,
    });

    return { success: true };
  }
}
