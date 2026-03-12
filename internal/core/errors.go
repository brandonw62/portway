// Copyright (C) 2024 Portway Contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// For commercial licensing, contact: licensing@portway.dev

package core

import "errors"

// Sentinel errors for domain-level conditions. Handlers map these to HTTP
// status codes; nothing outside the core package should produce raw HTTP errors.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when an operation would violate a uniqueness
	// constraint or produce a state conflict (e.g. duplicate slug).
	ErrConflict = errors.New("conflict")

	// ErrUnauthorized is returned when the caller lacks a valid identity.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the caller is authenticated but lacks
	// permission to perform the requested operation.
	ErrForbidden = errors.New("forbidden")

	// ErrValidation is returned when input fails business-rule validation.
	ErrValidation = errors.New("validation error")

	// ErrInternal is a catch-all for unexpected server-side failures that
	// should not leak internal detail to the caller.
	ErrInternal = errors.New("internal error")

	// ErrQuotaExceeded is returned when a provisioning request would exceed
	// the applicable resource quota.
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrApprovalRequired is returned when a provisioning request matches a
	// policy that requires explicit approval before proceeding.
	ErrApprovalRequired = errors.New("approval required")
)
