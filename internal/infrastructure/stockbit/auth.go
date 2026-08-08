package stockbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	loginPath   = "/login/v6/username"
	refreshPath = "/login/refresh"
	logoutPath  = "/logout"
)

// LoginRequest is the credentials the login endpoint expects.
type LoginRequest struct {
	PlayerID string `json:"player_id"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// TokenPair is one half of the access/refresh credential pair.
type TokenPair struct {
	Token     string `json:"token"`
	ExpiredAt string `json:"expired_at"`
}

// TokenData is the access+refresh pair returned by the auth endpoints.
type TokenData struct {
	Access  TokenPair `json:"access"`
	Refresh TokenPair `json:"refresh"`
}

// LoginUser is the account data returned by the login endpoint.
type LoginUser struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	Email              string    `json:"email"`
	Fullname           string    `json:"fullname"`
	Avatar             string    `json:"avatar"`
	Country            string    `json:"country"`
	SNS                SNS       `json:"sns"`
	HasPasswordBeenSet bool      `json:"has_password_been_set"`
	IsPhoneVerified    bool      `json:"is_phone_verified"`
	IsVerified         bool      `json:"is_verified"`
	Privilege          Privilege `json:"privilege"`
	WatchlistID        int64     `json:"watchlist_id"`
	Exchange           string    `json:"exchange"`
}

type SNS struct {
	Facebook bool `json:"facebook"`
	Apple    bool `json:"apple"`
	Google   bool `json:"google"`
}

type Privilege struct {
	Name string `json:"name"`
	Code int    `json:"code"`
}

// LoginResponse is the login response: data.login.{user,token_data} and
// data.support.
type LoginResponse struct {
	Message string `json:"message"`
	Data    struct {
		Login struct {
			User      LoginUser `json:"user"`
			TokenData TokenData `json:"token_data"`
		} `json:"login"`
		Support struct {
			ID string `json:"id"`
		} `json:"support"`
	} `json:"data"`
}

// RefreshResponse is the refresh response: data.{access,refresh}. The refresh
// token returned here replaces the one used to call Refresh (tokens rotate).
type RefreshResponse struct {
	Message string `json:"message"`
	Data    struct {
		Access  TokenPair `json:"access"`
		Refresh TokenPair `json:"refresh"`
	} `json:"data"`
}

func (c *Client) Login(ctx context.Context, params LoginRequest) (*LoginResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("stockbit: encode login request: %w", err)
	}
	var out LoginResponse
	if err := c.Post(ctx, loginPath, bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Refresh exchanges a refresh token for a fresh access/refresh pair.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	var out RefreshResponse
	err := c.do(ctx, http.MethodPost, refreshPath, nil, nil,
		map[string]string{"Authorization": "Bearer " + refreshToken}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// LogoutResponse is the logout response.
type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout invalidates the access token server-side.
func (c *Client) Logout(ctx context.Context, accessToken string) (*LogoutResponse, error) {
	var out LogoutResponse
	err := c.do(ctx, http.MethodPost, logoutPath, nil, nil,
		map[string]string{"Authorization": "Bearer " + accessToken}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
