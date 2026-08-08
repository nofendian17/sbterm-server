package stockbit

import "context"

const websocketKeyPath = "/auth/websocket/key"

// WebsocketKeyResponse is the response of the websocket key endpoint.
type WebsocketKeyResponse struct {
	Message string `json:"message"`
	Data    struct {
		Key string `json:"key"`
	} `json:"data"`
}

// GetWebsocketKey fetches the credential used to authenticate a Stockbit
// websocket connection. The access token is attached automatically.
func (c *Client) GetWebsocketKey(ctx context.Context) (*WebsocketKeyResponse, error) {
	var out WebsocketKeyResponse
	if err := c.Get(ctx, websocketKeyPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
