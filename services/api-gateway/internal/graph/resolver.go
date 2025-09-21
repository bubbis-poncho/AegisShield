package graph

import (
	"github.com/sirupsen/logrus"
	
	"aegisshield/services/api-gateway/internal/auth"
	"aegisshield/services/api-gateway/internal/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Services *services.ServiceClients
	Auth     *auth.Service
	Logger   *logrus.Logger
}