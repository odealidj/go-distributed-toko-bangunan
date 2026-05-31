import { checkoutMixed } from './scenarios/checkout-mixed.js';

export const options = {
  scenarios: {
    spike_checkout: {
      executor: 'ramping-vus',
      stages: [
        { duration: '10s', target: 2 },
        { duration: '10s', target: 20 },
        { duration: '20s', target: 20 },
        { duration: '10s', target: 2 },
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
