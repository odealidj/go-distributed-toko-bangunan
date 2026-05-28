package grpc

import (
	"context"
	"time"

	paymentv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/payment/v1"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentClient struct {
	client  paymentv1.PaymentServiceClient
	timeout time.Duration
	retries int
}

func NewPaymentClient(client paymentv1.PaymentServiceClient) *PaymentClient {
	return &PaymentClient{
		client:  client,
		timeout: 2 * time.Second,
		retries: 1,
	}
}

func (c *PaymentClient) CreatePayment(ctx context.Context, orderID string, amount int64, paymentMode, correlationID, causationID, idempotencyKey string) (model.Payment, error) {
	var response *paymentv1.CreatePaymentResponse
	err := c.call(ctx, func(callCtx context.Context) error {
		var err error
		response, err = c.client.CreatePayment(callCtx, &paymentv1.CreatePaymentRequest{
			Metadata:    paymentMetadata(correlationID, causationID, idempotencyKey),
			OrderId:     orderID,
			Amount:      amount,
			PaymentMode: paymentModeToProto(paymentMode),
		})
		return err
	})
	if err != nil {
		return model.Payment{}, paymentError(err)
	}
	return model.Payment{
		ID:     response.GetPaymentId(),
		Status: response.GetStatus(),
	}, nil
}

func (c *PaymentClient) call(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, c.timeout)
		lastErr = fn(callCtx)
		cancel()
		if lastErr == nil || !isTransient(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func paymentError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return model.ErrInvalidInput
	default:
		return err
	}
}

func paymentMetadata(correlationID, causationID, idempotencyKey string) *paymentv1.RequestMetadata {
	return &paymentv1.RequestMetadata{
		CorrelationId:  correlationID,
		CausationId:    causationID,
		IdempotencyKey: idempotencyKey,
	}
}

func paymentModeToProto(mode string) paymentv1.PaymentMode {
	switch mode {
	case model.PaymentModeFailure:
		return paymentv1.PaymentMode_PAYMENT_MODE_FAILURE
	case model.PaymentModeManual:
		return paymentv1.PaymentMode_PAYMENT_MODE_MANUAL
	default:
		return paymentv1.PaymentMode_PAYMENT_MODE_SUCCESS
	}
}
