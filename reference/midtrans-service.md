// midtrans.service.ts
import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as midtransClient from 'midtrans-client';

@Injectable()
export class MidtransService {
  private snap: midtransClient.Snap;

  constructor(private readonly config: ConfigService) {
    const isProduction =
      this.config.get<string>('MIDTRANS_IS_PRODUCTION') === 'true';
    const serverKey = this.config.get<string>('MIDTRANS_SERVER_KEY');
    const clientKey = this.config.get<string>('MIDTRANS_CLIENT_KEY');

    if (!serverKey || !clientKey) {
      throw new Error(
        'MIDTRANS_SERVER_KEY and MIDTRANS_CLIENT_KEY must be configured',
      );
    }

    this.snap = new midtransClient.Snap({
      isProduction,
      serverKey,
      clientKey,
    });
  }

  async createTransactionToken(params: {
   
    orderId: string;
    grossAmount: number;
    customer?: {
      firstName?: string;
      lastName?: string;
      email?: string;
      phone?: string;
    };
    items: {
      id: string;
      price: number;
      quantity: number;
      name: string;
    }[];
  }) {
    const transactionParams = {
      transaction_details: {
        order_id: params.orderId,
        gross_amount: params.grossAmount,
      },
      item_details: params.items,
      customer_details: params.customer,
    };
    const tx = await this.snap.createTransaction(transactionParams);
    // tx biasanya berisi { token, redirect_url }
    console.log("{tx midtrans service}")
    console.log({tx})
    return {
      snapToken: tx.token,
      redirectUrl: tx.redirect_url,
    };
  }
}
