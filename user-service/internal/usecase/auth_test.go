package usecase

import (
	"testing"
	"time"
	"user-service/internal/usecase/mocks"

	"go.uber.org/mock/gomock"
)

func setupTest(t *testing.T) (*gomock.Controller, *mocks.MockUserRepository, *mocks.MockSessionRepository, AuthUseCase) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)

	useCase := NewAuthUseCase(
		mockUserRepo, mockSessionRepo, "test-key",
		15*time.Minute, 24*time.Hour)

	return ctrl, mockUserRepo, mockSessionRepo, useCase
}
