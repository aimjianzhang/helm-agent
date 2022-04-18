package helm

import "helm-example/pkg/model"

func NewResponseReleaseError(key, cmdType, error string) *model.Packet {
	return &model.Packet{
		Key: key,
		Type: cmdType,
		Payload: error,
	}
}
