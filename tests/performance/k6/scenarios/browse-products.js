import { check } from 'k6';

import { defaultProductId, getProduct, listProducts } from '../lib/client.js';

export function browseProducts() {
  const listResponse = listProducts();
  check(listResponse, {
    'products list success envelope': (r) => r.json('success') === true,
  });

  const productResponse = getProduct(defaultProductId);
  check(productResponse, {
    'product detail success envelope': (r) => r.json('success') === true,
  });
}
