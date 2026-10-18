package localapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
)

// HubBackend serves app tokens on the hub.
type HubBackend struct{ Hub *hub.Hub }

func (b HubBackend) Models(ctx context.Context, tok string) ([]omnillm.ModelInfo, error) {
	return b.Hub.ModelsForApp(ctx, tok)
}

func (b HubBackend) Complete(ctx context.Context, tok string, req omnillm.Request) (omnillm.Stream, error) {
	return b.Hub.CompleteAsApp(ctx, tok, req)
}

// DeviceBackend serves "omnillm connect": one local token, calls go to the
// hub as Background.
type DeviceBackend struct {
	Device    *device.Device
	tokenHash [32]byte
}

// NewDeviceBackend makes a backend accepting exactly token.
func NewDeviceBackend(d *device.Device, token string) *DeviceBackend {
	return &DeviceBackend{Device: d, tokenHash: sha256.Sum256([]byte(token))}
}

func (b *DeviceBackend) check(tok string) error {
	h := sha256.Sum256([]byte(tok))
	if tok == "" || subtle.ConstantTimeCompare(h[:], b.tokenHash[:]) != 1 {
		return &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgInvalidToken}
	}
	return nil
}

func (b *DeviceBackend) Models(ctx context.Context, tok string) ([]omnillm.ModelInfo, error) {
	if err := b.check(tok); err != nil {
		return nil, err
	}
	return b.Device.Models(ctx)
}

func (b *DeviceBackend) Complete(ctx context.Context, tok string, req omnillm.Request) (omnillm.Stream, error) {
	if err := b.check(tok); err != nil {
		return nil, err
	}
	req.Caller = omnillm.Background
	return b.Device.Complete(ctx, req)
}
