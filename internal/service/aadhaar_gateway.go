package service

import (
	"context"
	"errors"
)

type AadhaarOTPRequest struct {
	PhoneNumber         string
	AadhaarReferenceKey string
}

type AadhaarOTPResponse struct {
	TransactionID string
	MobileLinked  bool
}

type AadhaarOTPVerificationRequest struct {
	TransactionID string
	OTP           string
}

type AadhaarOTPVerificationResponse struct {
	Verified bool
}

type AadhaarGateway interface {
	SendOTP(ctx context.Context, request AadhaarOTPRequest) (AadhaarOTPResponse, error)
	VerifyOTP(ctx context.Context, request AadhaarOTPVerificationRequest) (AadhaarOTPVerificationResponse, error)
}

type MockAadhaarGateway struct{}

func NewMockAadhaarGateway() *MockAadhaarGateway {
	return &MockAadhaarGateway{}
}

func (g *MockAadhaarGateway) SendOTP(ctx context.Context, request AadhaarOTPRequest) (AadhaarOTPResponse, error) {
	if request.PhoneNumber == "" || request.AadhaarReferenceKey == "" {
		return AadhaarOTPResponse{}, errors.New("phone number and aadhaar reference key are required")
	}

	return AadhaarOTPResponse{
		TransactionID: "mock-aadhaar-otp-transaction",
		MobileLinked:  true,
	}, nil
}

func (g *MockAadhaarGateway) VerifyOTP(ctx context.Context, request AadhaarOTPVerificationRequest) (AadhaarOTPVerificationResponse, error) {
	if request.TransactionID == "" || request.OTP == "" {
		return AadhaarOTPVerificationResponse{}, errors.New("transaction id and otp are required")
	}

	return AadhaarOTPVerificationResponse{Verified: true}, nil
}
