package broadcast

const _envelopeType = "poll"
const _envelopeVersion = "2022-06-05"

type Metadata struct {
	Type    string      `json:"type"`
	Version string      `json:"version"`
	Data    interface{} `json:"data"`
}

func CreateMetadata(data interface{}) Metadata {
	return Metadata{
		Type:    _envelopeType,
		Version: _envelopeVersion,
		Data:    data,
	}
}
