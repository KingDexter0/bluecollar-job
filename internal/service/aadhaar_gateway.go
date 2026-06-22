package service

import (
	"context"
	"errors"
)

type AadhaarOTPRequest struct {
	AadhaarNumber string
	PhoneNumber   string
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
	if request.AadhaarNumber == "" || request.PhoneNumber == "" {
		return AadhaarOTPResponse{}, errors.New("aadhaar number and phone number are required")
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
