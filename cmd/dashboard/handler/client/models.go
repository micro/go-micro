package client

type callRequest struct {
	Service  string `json:"service" binding:"required"`
	Version  string `json:"version"`
	Endpoint string `json:"endpoint" binding:"required"`
	Request  string `json:"request"`
	Timeout  int64  `json:"timeout"`
}
