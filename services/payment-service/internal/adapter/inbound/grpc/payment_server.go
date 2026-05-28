package grpc

import (
	"context"
	"errors"

	paymentv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/payment/v1"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	payment *usecase.PaymentUseCase
}

func NewPaymentServer(payment *usecase.PaymentUseCase) *PaymentServer {
	return &PaymentServer{payment: payment}
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	payment, err := s.payment.CreatePayment(ctx, model.CreatePaymentCommand{
		OrderID:        req.GetOrderId(),
		Amount:         req.GetAmount(),
		PaymentMode:    modeFromProto(req.GetPaymentMode()),
		IdempotencyKey: req.GetMetadata().GetIdempotencyKey(),
	})
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "order_id, amount, payment_mode, dan idempotency_key wajib diisi")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.CreatePaymentResponse{
		PaymentId: payment.ID,
		Status:    payment.Status,
	}, nil
}

func (s *PaymentServer) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.GetPaymentStatusResponse, error) {
	payment, err := s.payment.GetPaymentByID(ctx, req.GetPaymentId())
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "payment_id wajib diisi")
	}
	if errors.Is(err, model.ErrPaymentNotFound) {
		return nil, status.Error(codes.NotFound, "payment tidak ditemukan")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.GetPaymentStatusResponse{
		PaymentId: payment.ID,
		OrderId:   payment.OrderID,
		Status:    payment.Status,
		Amount:    payment.Amount,
	}, nil
}

func (s *PaymentServer) CancelPayment(ctx context.Context, req *paymentv1.CancelPaymentRequest) (*paymentv1.CancelPaymentResponse, error) {
	payment, err := s.payment.CancelPayment(ctx, model.CancelPaymentCommand{
		PaymentID: req.GetPaymentId(),
		OrderID:   req.GetOrderId(),
		Reason:    req.GetReason(),
	})
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "payment_id atau order_id wajib diisi")
	}
	if errors.Is(err, model.ErrPaymentNotFound) {
		return nil, status.Error(codes.NotFound, "payment tidak ditemukan")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.CancelPaymentResponse{
		PaymentId: payment.ID,
		Status:    payment.Status,
	}, nil
}

func modeFromProto(mode paymentv1.PaymentMode) string {
	switch mode {
	case paymentv1.PaymentMode_PAYMENT_MODE_SUCCESS:
		return model.PaymentModeSuccess
	case paymentv1.PaymentMode_PAYMENT_MODE_FAILURE:
		return model.PaymentModeFailure
	case paymentv1.PaymentMode_PAYMENT_MODE_MANUAL:
		return model.PaymentModeManual
	default:
		return ""
	}
}
