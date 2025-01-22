package events_test

import (
	"encoding/json"
	"errors"
	"gini-contract/chaincode/constants"
	"gini-contract/chaincode/events"
	"gini-contract/mocks"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitDenied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		address     string
		setupMock   func(*mocks.TransactionContext)
		shouldError bool
	}{
		{
			name:    "Success - Emit denied event",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - SetEvent error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(errors.New("failed to set event"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			if tt.setupMock != nil {
				tt.setupMock(ctx)
			}

			err := events.EmitDenied(ctx, tt.address)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify the event was set with correct data
				eventName, eventPayloadBytes := ctx.SetEventArgsForCall(0)
				require.Equal(t, constants.Denied, eventName)

				var eventPayload events.DeniedEvent
				err = json.Unmarshal(eventPayloadBytes, &eventPayload)
				require.NoError(t, err)
				require.Equal(t, tt.address, eventPayload.Address)
			}
		})
	}
}

func TestEmitAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		address     string
		setupMock   func(*mocks.TransactionContext)
		shouldError bool
	}{
		{
			name:    "Success - Emit allowed event",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - SetEvent error",
			address: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(errors.New("failed to set event"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			if tt.setupMock != nil {
				tt.setupMock(ctx)
			}

			err := events.EmitAllowed(ctx, tt.address)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				eventName, eventPayloadBytes := ctx.SetEventArgsForCall(0)
				require.Equal(t, constants.Allowed, eventName)

				var eventPayload events.AllowedEvent
				err = json.Unmarshal(eventPayloadBytes, &eventPayload)
				require.NoError(t, err)
				require.Equal(t, tt.address, eventPayload.Address)
			}
		})
	}
}

func TestEmitTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		from        string
		to          string
		value       string
		setupMock   func(*mocks.TransactionContext)
		shouldError bool
	}{
		{
			name:  "Success - Emit transfer event",
			from:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			to:    "2da4c4908a393a387b728206b18388bc529fa8d7",
			value: "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:  "Failure - SetEvent error",
			from:  "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			to:    "2da4c4908a393a387b728206b18388bc529fa8d7",
			value: "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(errors.New("failed to set event"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			if tt.setupMock != nil {
				tt.setupMock(ctx)
			}

			err := events.EmitTransfer(ctx, tt.from, tt.to, tt.value)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				eventName, eventPayloadBytes := ctx.SetEventArgsForCall(0)
				require.Equal(t, constants.Transfer, eventName)

				var eventPayload events.TransferEvent
				err = json.Unmarshal(eventPayloadBytes, &eventPayload)
				require.NoError(t, err)
				require.Equal(t, tt.from, eventPayload.From)
				require.Equal(t, tt.to, eventPayload.To)
				require.Equal(t, tt.value, eventPayload.Value)
			}
		})
	}
}

func TestEmitApproval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		owner       string
		spender     string
		value       string
		setupMock   func(*mocks.TransactionContext)
		shouldError bool
	}{
		{
			name:    "Success - Emit approval event",
			owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
			value:   "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - SetEvent error",
			owner:   "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			spender: "2da4c4908a393a387b728206b18388bc529fa8d7",
			value:   "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(errors.New("failed to set event"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			if tt.setupMock != nil {
				tt.setupMock(ctx)
			}

			err := events.EmitApproval(ctx, tt.owner, tt.spender, tt.value)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				eventName, eventPayloadBytes := ctx.SetEventArgsForCall(0)
				require.Equal(t, constants.Approval, eventName)

				var eventPayload events.ApprovalEvent
				err = json.Unmarshal(eventPayloadBytes, &eventPayload)
				require.NoError(t, err)
				require.Equal(t, tt.owner, eventPayload.Owner)
				require.Equal(t, tt.spender, eventPayload.Spender)
				require.Equal(t, tt.value, eventPayload.Value)
			}
		})
	}
}

func TestEmitMint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		account     string
		value       string
		setupMock   func(*mocks.TransactionContext)
		shouldError bool
	}{
		{
			name:    "Success - Emit mint event",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			value:   "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(nil)
			},
			shouldError: false,
		},
		{
			name:    "Failure - SetEvent error",
			account: "16f8ff33ef05bb24fb9a30fa79e700f57a496184",
			value:   "1000",
			setupMock: func(ctx *mocks.TransactionContext) {
				ctx.SetEventReturns(errors.New("failed to set event"))
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &mocks.TransactionContext{}
			if tt.setupMock != nil {
				tt.setupMock(ctx)
			}

			err := events.EmitMint(ctx, tt.account, tt.value)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				eventName, eventPayloadBytes := ctx.SetEventArgsForCall(0)
				require.Equal(t, constants.Mint, eventName)

				var eventPayload events.MintEvent
				err = json.Unmarshal(eventPayloadBytes, &eventPayload)
				require.NoError(t, err)
				require.Equal(t, tt.account, eventPayload.Account)
				require.Equal(t, tt.value, eventPayload.Value)
			}
		})
	}
}
