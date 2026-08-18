package stockbit

import "context"

const webSocketKeyPath = "/auth/websocket/key"

// WebSocketKeyResponse is the response of the websocket key endpoint.
type WebSocketKeyResponse struct {
	Message string `json:"message"`
	Data    struct {
		Key string `json:"key"`
	} `json:"data"`
}

// GetWebSocketKey fetches the credential used to authenticate a Stockbit
// websocket connection. The access token is attached automatically.
func (c *Client) GetWebSocketKey(ctx context.Context) (*WebSocketKeyResponse, error) {
	var out WebSocketKeyResponse
	if err := c.Get(ctx, webSocketKeyPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
