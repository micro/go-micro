package minecraft

import (
	"go-micro.dev/v4/api/client"
)

type Minecraft interface {
	Ping(*PingRequest) (*PingResponse, error)
}

func NewMinecraftService(token string) *MinecraftService {
	return &MinecraftService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type MinecraftService struct {
	client *client.Client
}

// Ping a minecraft server
func (t *MinecraftService) Ping(request *PingRequest) (*PingResponse, error) {

	rsp := &PingResponse{}
	return rsp, t.client.Call("minecraft", "Ping", request, rsp)

}

type PingRequest struct {
	// address of the server
	Address string `json:"address"`
}

type PingResponse struct {
	// Favicon in base64
	Favicon string `json:"favicon"`
	// Latency (ms) between us and the server (EU)
	Latency int32 `json:"latency"`
	// Max players ever
	MaxPlayers int32 `json:"max_players"`
	// Message of the day
	Motd string `json:"motd"`
	// Number of players online
	Players int32 `json:"players"`
	// Protocol number of the server
	Protocol int32 `json:"protocol"`
	// List of connected players
	Sample []PlayerSample `json:"sample"`
	// Version of the server
	Version string `json:"version"`
}

type PlayerSample struct {
	// name of the player
	Name string `json:"name"`
	// unique id of player
	Uuid string `json:"uuid"`
}
