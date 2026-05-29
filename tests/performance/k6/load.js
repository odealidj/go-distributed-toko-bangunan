import { browseProducts } from './scenarios/browse-products.js';
import { checkoutMixed } from './scenarios/checkout-mixed.js';

export const options = {
  scenarios: {
    browse: {
      executor: 'constant-vus',
      vus: 5,
      duration: '1m',
      exec: 'browseFlow',
    },
    checkout: {
      executor: 'constant-vus',
      vus: 3,
      duration: '1m',
      exec: 'checkoutFlow',
      startTime: '5s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{scenario:browse}': ['p(95)<500'],
    checkout_terminal_duration: ['p(95)<10000'],
  },
};

export function browseFlow() {
  browseProducts();
}

export function checkoutFlow() {
  checkoutMixed();
}
