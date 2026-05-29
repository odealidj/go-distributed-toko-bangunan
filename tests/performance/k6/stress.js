import { checkoutMixed } from './scenarios/checkout-mixed.js';

export const options = {
  scenarios: {
    stress_checkout: {
      executor: 'ramping-vus',
      stages: [
        { duration: '30s', target: 5 },
        { duration: '30s', target: 10 },
        { duration: '30s', target: 20 },
        { duration: '30s', target: 0 },
      ],
      exec: 'checkoutFlow',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    checkout_terminal_duration: ['p(95)<15000'],
  },
};

export function checkoutFlow() {
  checkoutMixed();
}
