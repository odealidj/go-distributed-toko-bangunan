import { check } from 'k6';

import { createOrder, defaultLowStockProductId, defaultProductId } from '../lib/client.js';
import { waitForTerminalStatus } from '../lib/polling.js';

export function checkoutMixed() {
  const selector = Math.random();
  let paymentMode = 'SUCCESS';
  let productId = defaultProductId;
  let quantity = 1;
  let expectedStatuses = ['CONFIRMED'];

  if (selector >= 0.7 && selector < 0.9) {
    paymentMode = 'FAILURE';
    expectedStatuses = ['CANCELLED'];
  } else if (selector >= 0.9) {
    paymentMode = 'SUCCESS';
    productId = defaultLowStockProductId;
    quantity = 10;
    expectedStatuses = ['REJECTED'];
  }

  const createResponse = createOrder({ paymentMode, productId, quantity });
  const orderId = createResponse.json('data.id');
  const terminal = waitForTerminalStatus(orderId, expectedStatuses);

  check(terminal, {
    'checkout mixed reaches expected terminal state': (result) => result.matchesExpectation,
  });
}
