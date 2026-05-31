import { checkoutMixed } from './scenarios/checkout-mixed.js';

export const options = {
  scenarios: {
    soak_checkout: {
      executor: 'constant-vus',
      vus: 2,
      duration: '5m',
      exec: 'checkoutFlow',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    checkout_terminal_duration: ['p(95)<12000'],
  },
};

export function checkoutFlow() {
  checkoutMixed();
}
