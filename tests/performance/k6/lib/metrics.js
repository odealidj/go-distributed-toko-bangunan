import { Counter, Rate, Trend } from 'k6/metrics';

export const checkoutTerminalDuration = new Trend('checkout_terminal_duration', true);
export const checkoutConfirmedTotal = new Counter('checkout_confirmed_total');
export const checkoutCancelledTotal = new Counter('checkout_cancelled_total');
export const checkoutRejectedTotal = new Counter('checkout_rejected_total');
export const checkoutTimeoutTotal = new Counter('checkout_timeout_total');
export const checkoutSuccessRate = new Rate('checkout_success_rate');
