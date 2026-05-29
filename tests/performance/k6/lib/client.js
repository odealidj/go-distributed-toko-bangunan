import http from 'k6/http';
import { check } from 'k6';

const orderBaseUrl = __ENV.K6_ORDER_BASE_URL || 'http://127.0.0.1:8080';
const catalogBaseUrl = __ENV.K6_CATALOG_BASE_URL || 'http://127.0.0.1:8081';

export const defaultProductId = __ENV.K6_PRODUCT_ID || 'prod_load_semen';
export const defaultLowStockProductId = __ENV.K6_LOW_STOCK_PRODUCT_ID || 'prod_load_low_stock';

function id(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1000000)}`;
}

function headers() {
  const requestId = id('req_k6');
  const correlationId = id('corr_k6');
  return {
    'Content-Type': 'application/json',
    'X-Request-Id': requestId,
    'X-Correlation-Id': correlationId,
  };
}

export function listProducts() {
  const response = http.get(`${catalogBaseUrl}/products?page=1&per_page=10`);
  check(response, {
    'GET /products status 200': (r) => r.status === 200,
  });
  return response;
}

export function getProduct(productId = defaultProductId) {
  const response = http.get(`${catalogBaseUrl}/products/${productId}`);
  check(response, {
    'GET /products/{id} status 200': (r) => r.status === 200,
  });
  return response;
}

export function createOrder({ paymentMode = 'SUCCESS', productId = defaultProductId, quantity = 1 } = {}) {
  const payload = JSON.stringify({
    customer: {
      name: 'K6 Tester',
      phone: '081234567890',
      address: 'Jl. Performance Test No. 1',
    },
    payment_mode: paymentMode,
    items: [
      {
        product_id: productId,
        quantity: quantity,
      },
    ],
  });

  const response = http.post(`${orderBaseUrl}/orders`, payload, { headers: headers() });
  check(response, {
    'POST /orders status 201': (r) => r.status === 201,
  });
  return response;
}

export function getOrder(orderId) {
  const response = http.get(`${orderBaseUrl}/orders/${orderId}`);
  check(response, {
    'GET /orders/{id} status 200': (r) => r.status === 200,
  });
  return response;
}
