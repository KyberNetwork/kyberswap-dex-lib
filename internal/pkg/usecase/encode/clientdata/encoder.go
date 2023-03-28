package clientdata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type (
	encoder struct {
		signer          ISigner
		clientDataKeyID string
	}

	clientDataWithIntegrityInfo struct {
		types.ClientData
		IntegrityInfo *integrityInfo
	}

	integrityInfo struct {
		KeyID     string
		Signature []byte
	}
)

func NewEncoder(signer ISigner, clientDataKeyID string) *encoder {
	return &encoder{
		clientDataKeyID: clientDataKeyID,
		signer:          signer,
	}
}

func (e *encoder) Encode(ctx context.Context, clientData types.ClientData) ([]byte, error) {
	info, err := e.signClientData(ctx, clientData)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("failed to sign clientData")
	}

	data := clientDataWithIntegrityInfo{
		ClientData:    clientData,
		IntegrityInfo: info,
	}

	return e.encode(data)
}

func (e *encoder) Decode(data []byte) (types.ClientData, error) {
	clientData, err := e.decode(data)
	if err != nil {
		return types.ClientData{}, nil
	}

	return clientData.ClientData, nil
}

func (e *encoder) signClientData(ctx context.Context, clientData types.ClientData) (*integrityInfo, error) {
	if e.signer == nil {
		return nil, ErrSignerNotFound
	}

	message := generateClientDataMsg(clientData)

	signature, err := e.signer.Sign(ctx, e.clientDataKeyID, message)
	if err != nil {
		return nil, err
	}

	return &integrityInfo{
		KeyID:     e.clientDataKeyID,
		Signature: signature,
	}, nil
}

func (e *encoder) encode(clientDataWithIntegrityInfo clientDataWithIntegrityInfo) ([]byte, error) {
	return json.Marshal(clientDataWithIntegrityInfo)
}

func (e *encoder) decode(data []byte) (clientDataWithIntegrityInfo, error) {
	var clientData clientDataWithIntegrityInfo

	if err := json.Unmarshal(data, &clientData); err != nil {
		return clientDataWithIntegrityInfo{}, err
	}

	return clientData, nil
}

// generateClientDataMsg transforms ClientData to a string for signing/verifying purpose
func generateClientDataMsg(clientData types.ClientData) string {
	return fmt.Sprintf("source-%s_amountInUSD-%s_amountOutUSD-%s_referral-%s", clientData.Source, clientData.AmountInUSD, clientData.AmountOutUSD, clientData.Referral)
}
