package clientdata

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/mocks/usecase/encode/clientdata"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
)

func TestEncoder_Encode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		clientData    types.ClientData
		prepare       func(ctrl *gomock.Controller) encoder
		expectedData  string
		expectedError error
	}{
		{
			name: "it should encode successfully when signer is nil",
			clientData: types.ClientData{
				Source:       "kyberswap",
				AmountInUSD:  "1000000",
				AmountOutUSD: "999999",
				Referral:     "",
				Flags:        12,
			},
			prepare: func(_ *gomock.Controller) encoder {
				return encoder{}
			},
			expectedData: `{"Source":"kyberswap","AmountInUSD":"1000000","AmountOutUSD":"999999","Referral":"","Flags":12,"IntegrityInfo":null}`,
		},
		{
			name: "it should encode successfully when signer sign failed",
			clientData: types.ClientData{
				Source:       "kyberswap",
				AmountInUSD:  "1000000",
				AmountOutUSD: "999999",
				Referral:     "",
				Flags:        12,
			},
			prepare: func(ctrl *gomock.Controller) encoder {
				signer := clientdata.NewMockISigner(ctrl)
				signer.EXPECT().Sign(gomock.Any(), "clientDataKeyID", "source-kyberswap_amountInUSD-1000000_amountOutUSD-999999_referral-").
					Return(nil, errors.New("something went wrong"))

				return encoder{
					clientDataKeyID: "clientDataKeyID",
					signer:          signer,
				}
			},
			expectedData: `{"Source":"kyberswap","AmountInUSD":"1000000","AmountOutUSD":"999999","Referral":"","Flags":12,"IntegrityInfo":null}`,
		},
		{
			name: "it should encode successfully when signer sign successfully",
			clientData: types.ClientData{
				Source:       "kyberswap",
				AmountInUSD:  "1000000",
				AmountOutUSD: "999999",
				Referral:     "",
				Flags:        12,
			},
			prepare: func(ctrl *gomock.Controller) encoder {
				signer := clientdata.NewMockISigner(ctrl)
				signer.EXPECT().Sign(gomock.Any(), "clientDataKeyID", "source-kyberswap_amountInUSD-1000000_amountOutUSD-999999_referral-").
					Return([]byte{1, 2, 3}, nil)

				return encoder{
					clientDataKeyID: "clientDataKeyID",
					signer:          signer,
				}
			},
			expectedData: `{"Source":"kyberswap","AmountInUSD":"1000000","AmountOutUSD":"999999","Referral":"","Flags":12,"IntegrityInfo":{"KeyID":"clientDataKeyID","Signature":"AQID"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			encoder := tc.prepare(ctrl)

			data, err := encoder.Encode(ctx, tc.clientData)

			assert.ErrorIs(t, err, tc.expectedError)
			assert.Equal(t, tc.expectedData, string(data))
		})
	}
}

func TestEncoder_Decode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		data               string
		expectedClientData types.ClientData
		expectedError      error
	}{
		{
			name: "it should encode successfully when signer is nil",
			data: `{"Source":"kyberswap","AmountInUSD":"1000000","AmountOutUSD":"999999","Referral":"","Flags":12,"IntegrityInfo":null}`,
			expectedClientData: types.ClientData{
				Source:       "kyberswap",
				AmountInUSD:  "1000000",
				AmountOutUSD: "999999",
				Referral:     "",
				Flags:        12,
			},
		},
		{
			name: "it should encode successfully when signer sign successfully",
			expectedClientData: types.ClientData{
				Source:       "kyberswap",
				AmountInUSD:  "1000000",
				AmountOutUSD: "999999",
				Referral:     "",
				Flags:        12,
			},
			data: `{"Source":"kyberswap","AmountInUSD":"1000000","AmountOutUSD":"999999","Referral":"","Flags":12,"IntegrityInfo":{"KeyID":"clientDataKeyID","Signature":"AQID"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := encoder{}

			clientData, err := e.Decode([]byte(tc.data))

			assert.ErrorIs(t, err, tc.expectedError)
			assert.Equal(t, tc.expectedClientData, clientData)
		})
	}
}
