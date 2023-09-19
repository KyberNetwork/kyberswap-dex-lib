package executor

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
)

func PackPositiveSlippageFeeData(inputs PositiveSlippageFeeData) ([]byte, error) {
	// partnerPSInfor: [partnerReceiver (160 bits) + partnerPercent (96 bits)]
	// expectedReturnAmount: [minimumPSAmount (128 bits) + expectedReturnAmount (128 bits)]
	return pack.Pack(
		inputs.PartnerReceiver,
		inputs.PartnerPercent,
		inputs.MinimumPSAmount,
		inputs.ExpectedReturnAmount,
	)
}

func UnpackPositiveSlippageFeeData(bytes []byte) (PositiveSlippageFeeData, error) {
	var psData PositiveSlippageFeeData
	var startByte int

	psData.PartnerReceiver, startByte = pack.ReadUInt160(bytes, startByte)
	psData.PartnerPercent, startByte = pack.ReadUInt96(bytes, startByte)
	psData.MinimumPSAmount, startByte = pack.ReadBigInt(bytes, startByte)
	psData.ExpectedReturnAmount, _ = pack.ReadBigInt(bytes, startByte)

	return psData, nil
}
