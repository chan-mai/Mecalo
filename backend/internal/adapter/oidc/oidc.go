// OpenID ConnectのAuthorization Code + PKCE
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi"
	"github.com/chan-mai/mecalo/backend/internal/app"
)

type Config struct {
	// ログイン画面の表示名
	Name     string
	Issuer   string
	ClientID string
	// public clientでは空
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type Provider struct {
	name     string
	oauth    oauth2.Config
	verifier *gooidc.IDTokenVerifier
}

var _ httpapi.OIDCProvider = (*Provider)(nil)

func New(ctx context.Context, cfg Config) (*Provider, error) {
	ctx = gooidc.ClientContext(context.WithoutCancel(ctx), &http.Client{Timeout: 15 * time.Second})
	p, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{gooidc.ScopeOpenID, "profile"}
	}
	return &Provider{
		name: cfg.Name,
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     p.Endpoint(),
			Scopes:       scopes,
		},
		verifier: p.Verifier(&gooidc.Config{ClientID: cfg.ClientID}),
	}, nil
}

func (p *Provider) Name() string {
	return p.name
}

func (p *Provider) Begin(_ context.Context) (string, httpapi.OIDCFlow, error) {
	state, err := random()
	if err != nil {
		return "", httpapi.OIDCFlow{}, err
	}
	nonce, err := random()
	if err != nil {
		return "", httpapi.OIDCFlow{}, err
	}
	flow := httpapi.OIDCFlow{State: state, Nonce: nonce, Verifier: oauth2.GenerateVerifier()}
	url := p.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(flow.Verifier), gooidc.Nonce(nonce))
	return url, flow, nil
}

func (p *Provider) Finish(ctx context.Context, code string, flow httpapi.OIDCFlow) (app.ExternalIdentity, error) {
	tok, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(flow.Verifier))
	if err != nil {
		return app.ExternalIdentity{}, fmt.Errorf("oidc token exchange: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return app.ExternalIdentity{}, errors.New("oidc response has no id_token")
	}
	idt, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return app.ExternalIdentity{}, fmt.Errorf("oidc id_token: %w", err)
	}
	if idt.Nonce != flow.Nonce {
		return app.ExternalIdentity{}, errors.New("oidc nonce mismatch")
	}
	var claims struct {
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idt.Claims(&claims); err != nil {
		return app.ExternalIdentity{}, err
	}
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	return app.ExternalIdentity{Issuer: idt.Issuer, Subject: idt.Subject, DisplayName: name}, nil
}

func random() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
