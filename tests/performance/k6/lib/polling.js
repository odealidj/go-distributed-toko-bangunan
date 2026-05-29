import { sleep } from 'k6';

import { getOrder } from './client.js';
import {
  checkoutCancelledTotal,
  checkoutConfirmedTotal,
  checkoutRejectedTotal,
  checkoutSuccessRate,
  checkoutTerminalDuration,
  checkoutTimeoutTotal,
} from './metrics.js';

const terminalStatuses = new Set(['CONFIRMED', 'CANCELLED', 'REJECTED']);

export function waitForTerminalStatus(orderId, expectedStatuses, timeoutMs = 30000, intervalSeconds = 1) {
  const startedAt = Date.now();

  while (Date.now() - startedAt < timeoutMs) {
    const response = getOrder(orderId);
    const order = response.json('data');
    const status = order.status;

    if (terminalStatuses.has(status)) {
      const durationMs = Date.now() - startedAt;
      checkoutTerminalDuration.add(durationMs);

      if (status === 'CONFIRMED') {
        checkoutConfirmedTotal.add(1);
        checkoutSuccessRate.add(true);
      } else if (status === 'CANCELLED') {
        checkoutCancelledTotal.add(1);
        checkoutSuccessRate.add(false);
      } else {
        checkoutRejectedTotal.add(1);
        checkoutSuccessRate.add(false);
      }

      return {
        status,
        durationMs,
        order,
        response,
        matchesExpectation: expectedStatuses.includes(status),
      };
    }

    sleep(intervalSeconds);
  }

  checkoutTimeoutTotal.add(1);
  checkoutSuccessRate.add(false);
  return {
    status: 'TIMEOUT',
    durationMs: Date.now() - startedAt,
    matchesExpectation: false,
  };
}
