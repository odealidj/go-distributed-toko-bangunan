import { browseProducts } from './scenarios/browse-products.js';
import { checkoutSuccess } from './scenarios/checkout-success.js';

export const options = {
  vus: 1,
  iterations: 3,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
    checkout_terminal_duration: ['p(95)<10000'],
  },
};

export default function () {
  browseProducts();
  checkoutSuccess();
}
