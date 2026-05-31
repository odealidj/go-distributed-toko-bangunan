import { check } from 'k6';

import { createOrder, defaultProductId } from '../lib/client.js';
import { waitForTerminalStatus } from '../lib/polling.js';

export function checkoutSuccess() {
  const createResponse = createOrder({
    paymentMode: 'SUCCESS',
    productId: defaultProductId,
    quantity: 1,
  });

  const orderId = createResponse.json('data.id');
  const terminal = waitForTerminalStatus(orderId, ['CONFIRMED']);

  check(terminal, {
    'checkout success reaches CONFIRMED': (result) => result.matchesExpectation,
  });
}
