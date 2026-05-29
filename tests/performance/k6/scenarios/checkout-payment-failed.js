import { check } from 'k6';

import { createOrder, defaultProductId } from '../lib/client.js';
import { waitForTerminalStatus } from '../lib/polling.js';

export function checkoutPaymentFailed() {
  const createResponse = createOrder({
    paymentMode: 'FAILURE',
    productId: defaultProductId,
    quantity: 1,
  });

  const orderId = createResponse.json('data.id');
  const terminal = waitForTerminalStatus(orderId, ['CANCELLED']);

  check(terminal, {
    'checkout failure reaches CANCELLED': (result) => result.matchesExpectation,
  });
}
